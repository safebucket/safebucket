package helpers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/alexedwards/argon2id"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/hkdf"
)

type tokenConfig struct {
	audience      string
	provider      string
	mfa           *bool
	expiryMinutes int
	challengeID   *uuid.UUID
	sid           string
}

func boolPtr(b bool) *bool {
	return &b
}

func generateJTI(audience string) string {
	switch audience {
	case configuration.AudienceAccessToken:
		return fmt.Sprintf("access:%s", uuid.New().String())
	case configuration.AudienceRefreshToken:
		return fmt.Sprintf("refresh:%s", uuid.New().String())
	default:
		return ""
	}
}

var jweSalt = sha256.Sum256([]byte("safebucket/jwe/v1/salt"))

func deriveJWEKey(secret string) ([]byte, error) {
	reader := hkdf.New(sha256.New, []byte(secret), jweSalt[:], []byte("safebucket-jwe-v1"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("hkdf: %w", err)
	}
	return key, nil
}

func encryptJWE(secret, jws string) (string, error) {
	key, err := deriveJWEKey(secret)
	if err != nil {
		return "", err
	}
	enc, err := jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{Algorithm: jose.DIRECT, Key: key},
		nil,
	)
	if err != nil {
		return "", err
	}
	obj, err := enc.Encrypt([]byte(jws))
	if err != nil {
		return "", err
	}
	return obj.CompactSerialize()
}

func decryptJWE(secret, jweCompact string) (string, error) {
	key, err := deriveJWEKey(secret)
	if err != nil {
		return "", err
	}
	obj, err := jose.ParseEncrypted(
		jweCompact,
		[]jose.KeyAlgorithm{jose.DIRECT},
		[]jose.ContentEncryption{jose.A256GCM},
	)
	if err != nil {
		return "", errors.New("invalid token")
	}
	plaintext, err := obj.Decrypt(key)
	if err != nil {
		return "", errors.New("invalid token")
	}
	return string(plaintext), nil
}

func signAndEncryptToken(jwtSecret string, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jws, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}
	return encryptJWE(jwtSecret, jws)
}

func decryptAndParseJWT(jwtSecret string, tokenString string, claims jwt.Claims) error {
	jws, err := decryptJWE(jwtSecret, tokenString)
	if err != nil {
		return errors.New("invalid token")
	}
	_, err = jwt.ParseWithClaims(jws, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	return err
}

func stripBearer(tokenString string) (string, error) {
	if !strings.HasPrefix(tokenString, "Bearer ") {
		return "", errors.New("invalid token")
	}
	return strings.TrimPrefix(tokenString, "Bearer "), nil
}

func newRegisteredClaims(
	audience string,
	expiryMinutes int,
) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Issuer:   configuration.AppName,
		Audience: jwt.ClaimStrings{audience},
		IssuedAt: &jwt.NumericDate{Time: time.Now()},
		ExpiresAt: &jwt.NumericDate{
			Time: time.Now().Add(
				time.Minute * time.Duration(expiryMinutes),
			),
		},
	}
}

func createToken(jwtSecret string, user *models.User, config tokenConfig) (string, error) {
	jti := generateJTI(config.audience)

	claims := models.UserClaims{
		Email:    user.Email,
		UserID:   user.ID,
		Role:     user.Role,
		Provider: config.provider,
		SID:      config.sid,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    configuration.AppName,
			Audience:  jwt.ClaimStrings{config.audience},
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute * time.Duration(config.expiryMinutes))},
		},
	}

	if config.mfa != nil {
		claims.MFA = *config.mfa
	}

	if config.challengeID != nil {
		claims.ChallengeID = config.challengeID
	}

	return signAndEncryptToken(jwtSecret, claims)
}

func ParseToken(
	jwtSecret string,
	tokenString string,
	requireBearer bool,
) (models.UserClaims, error) {
	if requireBearer {
		var err error
		tokenString, err = stripBearer(tokenString)
		if err != nil {
			return models.UserClaims{}, err
		}
	}

	claims := &models.UserClaims{}
	if err := decryptAndParseJWT(jwtSecret, tokenString, claims); err != nil {
		return models.UserClaims{}, errors.New("invalid token")
	}

	if len(claims.Audience) != 1 {
		return models.UserClaims{}, errors.New("invalid token audience")
	}

	return *claims, nil
}

func NewShareAccessToken(
	jwtSecret string,
	shareID uuid.UUID,
) (string, error) {
	claims := models.ShareClaims{
		ShareID: shareID,
		RegisteredClaims: newRegisteredClaims(
			configuration.AudienceShareAccess,
			configuration.ShareTokenExpiry,
		),
	}
	return signAndEncryptToken(jwtSecret, claims)
}

func ParseShareToken(
	jwtSecret string,
	tokenString string,
) (models.ShareClaims, error) {
	raw, err := stripBearer(tokenString)
	if err != nil {
		return models.ShareClaims{}, err
	}

	claims := &models.ShareClaims{}
	if parseErr := decryptAndParseJWT(jwtSecret, raw, claims); parseErr != nil {
		return models.ShareClaims{}, errors.New("invalid token")
	}

	if len(claims.Audience) != 1 ||
		claims.Audience[0] != configuration.AudienceShareAccess {
		return models.ShareClaims{}, errors.New("invalid token audience")
	}

	return *claims, nil
}

func CreateHash(password string) (string, error) {
	argonParams := argon2id.Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  32,
		KeyLength:   32,
	}
	hash, err := argon2id.CreateHash(password, &argonParams)
	if err != nil {
		return "", errors.New("can not create hash password")
	}

	return hash, nil
}

func NewAccessToken(jwtSecret string, user *models.User, provider string, sid string) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      configuration.AudienceAccessToken,
		provider:      provider,
		mfa:           boolPtr(user.HasMFAEnabled()),
		expiryMinutes: configuration.AccessTokenExpiry,
		sid:           sid,
	})
}

func NewRefreshToken(jwtSecret string, user *models.User, provider string, sid string) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      configuration.AudienceRefreshToken,
		provider:      provider,
		mfa:           boolPtr(user.HasMFAEnabled()),
		expiryMinutes: configuration.RefreshTokenExpiry,
		sid:           sid,
	})
}

func ParseRefreshToken(jwtSecret string, refreshToken string) (models.UserClaims, error) {
	claims, err := ParseToken(jwtSecret, refreshToken, false)
	if err != nil {
		return models.UserClaims{}, errors.New("invalid refresh token")
	}

	if claims.AudienceString() != configuration.AudienceRefreshToken {
		return models.UserClaims{}, errors.New("invalid refresh token audience")
	}

	return claims, nil
}

func GetUserClaims(c context.Context) (models.UserClaims, error) {
	value, ok := c.Value(models.UserClaimKey{}).(models.UserClaims)
	if !ok {
		return models.UserClaims{}, errors.New("invalid user claims")
	}
	return value, nil
}

func GenerateSecret() (string, error) {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const secretLength = 6
	secret := make([]byte, secretLength)
	for i := range secret {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		secret[i] = charset[n.Int64()]
	}
	return string(secret), nil
}

func NewRestrictedAccessToken(
	jwtSecret string,
	user *models.User,
	audience string,
	mfaVerified bool,
	challengeID *uuid.UUID,
) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      audience,
		provider:      string(user.ProviderType),
		mfa:           boolPtr(mfaVerified),
		expiryMinutes: configuration.MFATokenExpiry,
		challengeID:   challengeID,
	})
}
