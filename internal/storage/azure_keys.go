package storage

import (
	"context"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"go.uber.org/zap"
)

func resolveAzureCredentials(config *models.AzureConfiguration) (string, azcore.TokenCredential) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		zap.L().Fatal("Failed to create Azure default credential", zap.Error(err))
	}

	return listAzurePrimaryAccountKey(config, cred), cred
}

func listAzurePrimaryAccountKey(config *models.AzureConfiguration, cred azcore.TokenCredential) string {
	client, err := armstorage.NewAccountsClient(config.SubscriptionID, cred, nil)
	if err != nil {
		zap.L().Fatal("Failed to create Azure storage accounts client", zap.Error(err))
	}

	resp, err := client.ListKeys(context.Background(), config.ResourceGroup, config.AccountName, nil)
	if err != nil {
		zap.L().Fatal("Failed to list Azure storage account keys", zap.Error(err))
	}

	if len(resp.Keys) == 0 || resp.Keys[0].Value == nil {
		zap.L().Fatal("Azure storage account has no keys")
	}

	return *resp.Keys[0].Value
}
