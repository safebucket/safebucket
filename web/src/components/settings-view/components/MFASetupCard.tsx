import { useState } from "react";
import { useTranslation } from "react-i18next";
import { format } from "date-fns";
import { fr, enUS } from "date-fns/locale";
import QRCode from "react-qr-code";
import {
  Shield,
  ShieldCheck,
  Copy,
  CheckCircle,
  AlertCircle,
  RefreshCw,
  Plus,
  Trash2,
  Star,
  Smartphone,
} from "lucide-react";

import type {
  IUser,
  IMFADevice,
  IMFADeviceSetupResponse,
} from "@/components/auth-view/types/session";
import { authCookies } from "@/lib/auth-service";
import { useQueryClient } from "@tanstack/react-query";
import {
  useRequestMFAResetMutation,
  useVerifyMFAResetMutation,
  useMFADevices,
  useAddMFADeviceMutation,
  useVerifyMFADeviceMutation,
  useRemoveMFADeviceMutation,
} from "@/queries/user";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";

interface MFASetupCardProps {
  user: IUser;
}

type SetupStep = "name" | "qr" | "verify" | "success";
type ResetStep = "idle" | "password" | "email_sent" | "verify_code" | "success";

export function MFASetupCard({ user }: MFASetupCardProps) {
  const { t, i18n } = useTranslation();
  const queryClient = useQueryClient();
  const dateLocale = i18n.language === "fr" ? fr : enUS;

  // Fetch devices
  const { data: devicesData, isLoading: devicesLoading } = useMFADevices(
    user.id
  );

  // Setup state
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [setupStep, setSetupStep] = useState<SetupStep>("name");
  const [setupData, setSetupData] = useState<IMFADeviceSetupResponse | null>(
    null
  );
  const [deviceName, setDeviceName] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [secretCopied, setSecretCopied] = useState(false);

  // Delete confirmation
  const [deleteDeviceId, setDeleteDeviceId] = useState<string | null>(null);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // Reset state
  const [isResetDialogOpen, setIsResetDialogOpen] = useState(false);
  const [resetStep, setResetStep] = useState<ResetStep>("idle");
  const [resetPassword, setResetPassword] = useState("");
  const [resetCode, setResetCode] = useState("");
  const [challengeId, setChallengeId] = useState<string | null>(null);
  const [resetError, setResetError] = useState<string | null>(null);

  const addDeviceMutation = useAddMFADeviceMutation(user.id);
  const verifyDeviceMutation = useVerifyMFADeviceMutation(
    user.id,
    setupData?.device_id || ""
  );
  const removeDeviceMutation = useRemoveMFADeviceMutation(user.id);
  const requestResetMutation = useRequestMFAResetMutation(user.id);
  const verifyResetMutation = useVerifyMFAResetMutation(
    user.id,
    challengeId || ""
  );

  const devices = devicesData?.devices || [];
  const mfaEnabled = devicesData?.mfa_enabled || user.mfa_enabled;
  const deviceCount = devicesData?.device_count || 0;
  const maxDevices = devicesData?.max_devices || 5;

  const handleOpenAddDialog = () => {
    setIsDialogOpen(true);
    setSetupStep("name");
    setDeviceName("");
    setSetupData(null);
    setCode("");
    setError(null);
  };

  const handleStartSetup = async () => {
    if (!deviceName.trim()) {
      setError(t("auth.mfa.error_device_name_required"));
      return;
    }

    setError(null);
    addDeviceMutation.mutate(deviceName.trim(), {
      onSuccess: (response) => {
        setSetupData(response);
        setSetupStep("qr");
      },
      onError: (err) => {
        if (err.message.includes("MFA_DEVICE_NAME_EXISTS")) {
          setError(t("auth.mfa.error_device_name_exists"));
        } else if (err.message.includes("MAX_MFA_DEVICES_REACHED")) {
          setError(t("auth.mfa.error_max_devices"));
        } else {
          setError(t("auth.mfa.setup_error"));
        }
      },
    });
  };

  const handleVerifyCode = async () => {
    if (code.length !== 6) {
      setError(t("auth.mfa.error_code_length"));
      return;
    }

    setError(null);
    verifyDeviceMutation.mutate(code, {
      onSuccess: (response) => {
        // Save the new tokens with updated MFA claim
        const tokenResponse = response as {
          access_token?: string;
          refresh_token?: string;
        };
        if (tokenResponse.access_token) {
          authCookies.setAccessToken(tokenResponse.access_token);
        }
        if (tokenResponse.refresh_token) {
          authCookies.setRefreshToken(tokenResponse.refresh_token);
        }
        setSetupStep("success");
        queryClient.invalidateQueries({ queryKey: ["users", user.id] });
      },
      onError: () => {
        setError(t("auth.mfa.verify_error"));
      },
    });
  };

  const handleCopySecret = async () => {
    if (setupData?.secret) {
      await navigator.clipboard.writeText(setupData.secret);
      setSecretCopied(true);
      setTimeout(() => setSecretCopied(false), 2000);
    }
  };

  const handleCloseDialog = () => {
    setIsDialogOpen(false);
    setSetupStep("name");
    setSetupData(null);
    setDeviceName("");
    setCode("");
    setError(null);
  };

  const handleDeleteDevice = (deviceId: string) => {
    setDeleteDeviceId(deviceId);
    setDeletePassword("");
    setDeleteError(null);
  };

  const handleCloseDeleteDialog = () => {
    setDeleteDeviceId(null);
    setDeletePassword("");
    setDeleteError(null);
  };

  const handleConfirmDelete = () => {
    if (deleteDeviceId && deletePassword) {
      setDeleteError(null);
      removeDeviceMutation.mutate(
        { deviceId: deleteDeviceId, password: deletePassword },
        {
          onSuccess: () => {
            handleCloseDeleteDialog();
          },
          onError: (err) => {
            if (err.message.includes("INVALID_PASSWORD")) {
              setDeleteError(t("auth.mfa.delete_invalid_password"));
            } else {
              setDeleteError(t("auth.mfa.delete_error"));
            }
          },
        }
      );
    }
  };

  const handleSetPrimary = async (deviceId: string) => {
    try {
      const { api } = await import("@/lib/api");
      await api.patch(`/users/${user.id}/mfa/devices/${deviceId}`, {
        is_primary: true,
      });
      queryClient.invalidateQueries({
        queryKey: ["users", user.id, "mfa", "devices"],
      });
    } catch {
      // Error handled by toast in api
    }
  };

  // Reset handlers
  const handleOpenResetDialog = () => {
    setIsResetDialogOpen(true);
    setResetStep("password");
    setResetPassword("");
    setResetCode("");
    setResetError(null);
    setChallengeId(null);
  };

  const handleCloseResetDialog = () => {
    setIsResetDialogOpen(false);
    setResetStep("idle");
    setResetPassword("");
    setResetCode("");
    setResetError(null);
    setChallengeId(null);
  };

  const handleRequestReset = async () => {
    setResetError(null);
    requestResetMutation.mutate(resetPassword, {
      onSuccess: (response) => {
        setChallengeId(response.challenge_id);
        setResetStep("email_sent");
      },
      onError: (err) => {
        if (err.message.includes("INVALID_PASSWORD")) {
          setResetError(t("auth.mfa.reset_invalid_password"));
        } else {
          setResetError(t("auth.mfa.reset_request_error"));
        }
      },
    });
  };

  const handleVerifyReset = async () => {
    setResetError(null);
    verifyResetMutation.mutate(resetCode, {
      onSuccess: () => {
        setResetStep("success");
        queryClient.invalidateQueries({ queryKey: ["users", user.id] });
        queryClient.invalidateQueries({
          queryKey: ["users", user.id, "mfa", "devices"],
        });
      },
      onError: (err) => {
        if (err.message.includes("WRONG_CODE")) {
          setResetError(t("auth.mfa.reset_wrong_code"));
        } else if (err.message.includes("CHALLENGE_EXPIRED")) {
          setResetError(t("auth.mfa.reset_challenge_expired"));
        } else if (err.message.includes("CHALLENGE_LOCKED")) {
          setResetError(t("auth.mfa.reset_challenge_locked"));
        } else {
          setResetError(t("auth.mfa.reset_verify_error"));
        }
      },
    });
  };

  const DeviceRow = ({ device }: { device: IMFADevice }) => (
    <div className="flex items-center justify-between rounded-lg border p-4">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
          <Smartphone className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-medium">{device.name}</span>
            {device.is_primary && (
              <Badge variant="secondary" className="text-xs">
                <Star className="mr-1 h-3 w-3" />
                {t("auth.mfa.primary")}
              </Badge>
            )}
          </div>
          <div className="text-muted-foreground text-xs">
            {t("auth.mfa.added_on", {
              date: format(new Date(device.created_at), "PPP", { locale: dateLocale }),
            })}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {!device.is_primary && device.is_verified && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => handleSetPrimary(device.id)}
          >
            <Star className="mr-1 h-4 w-4" />
            {t("auth.mfa.set_primary")}
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="text-destructive hover:text-destructive"
          onClick={() => handleDeleteDevice(device.id)}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            {mfaEnabled ? (
              <ShieldCheck className="h-4 w-4 text-green-600" />
            ) : (
              <Shield className="h-4 w-4" />
            )}
            {t("auth.mfa.devices_title")}
          </CardTitle>
          <CardDescription>
            {t("auth.mfa.devices_description", {
              count: deviceCount,
              max: maxDevices,
            })}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {devicesLoading ? (
            <div className="text-muted-foreground text-sm">
              {t("common.loading")}
            </div>
          ) : devices.length > 0 ? (
            <>
              <div className="space-y-3">
                {devices.map((device) => (
                  <DeviceRow key={device.id} device={device} />
                ))}
              </div>

              <div className="flex items-center justify-between pt-4 border-t">
                <Button
                  variant="outline"
                  onClick={handleOpenAddDialog}
                  disabled={deviceCount >= maxDevices}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  {t("auth.mfa.add_device")}
                </Button>
                <Button variant="ghost" onClick={handleOpenResetDialog}>
                  <RefreshCw className="mr-2 h-4 w-4" />
                  {t("auth.mfa.reset_all")}
                </Button>
              </div>
            </>
          ) : (
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <Badge variant="secondary">{t("auth.mfa.not_enabled")}</Badge>
                </div>
                <p className="text-muted-foreground text-sm">
                  {t("auth.mfa.not_enabled_description")}
                </p>
              </div>
              <Button onClick={handleOpenAddDialog}>
                {t("auth.mfa.setup_button")}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Add Device Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent
          className="sm:max-w-md"
          showCloseButton={setupStep !== "success"}
        >
          {setupStep === "name" && (
            <>
              <DialogHeader>
                <DialogTitle>{t("auth.mfa.add_device_title")}</DialogTitle>
                <DialogDescription>
                  {t("auth.mfa.add_device_description")}
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4">
                {error && (
                  <div className="flex items-center gap-2 rounded-md bg-red-50 p-3 text-sm text-red-600">
                    <AlertCircle className="h-4 w-4" />
                    {error}
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="device-name">
                    {t("auth.mfa.device_name_label")}
                  </Label>
                  <Input
                    id="device-name"
                    value={deviceName}
                    onChange={(e) => setDeviceName(e.target.value)}
                    placeholder={t("auth.mfa.device_name_placeholder")}
                    disabled={addDeviceMutation.isPending}
                  />
                </div>
              </div>

              <DialogFooter className="sm:justify-between">
                <Button variant="outline" onClick={handleCloseDialog}>
                  {t("common.cancel")}
                </Button>
                <Button
                  onClick={handleStartSetup}
                  disabled={!deviceName.trim() || addDeviceMutation.isPending}
                >
                  {addDeviceMutation.isPending
                    ? t("common.loading")
                    : t("auth.continue")}
                </Button>
              </DialogFooter>
            </>
          )}

          {setupStep === "qr" && setupData && (
            <>
              <DialogHeader>
                <DialogTitle>{t("auth.mfa.qr_code_title")}</DialogTitle>
                <DialogDescription>
                  {t("auth.mfa.qr_code_instruction")}
                </DialogDescription>
              </DialogHeader>

              <div className="flex flex-col items-center space-y-4">
                <div className="rounded-lg bg-white p-4">
                  <QRCode value={setupData.qr_code_uri} size={180} />
                </div>

                <div className="w-full space-y-2">
                  <p className="text-sm font-medium">
                    {t("auth.mfa.manual_entry_title")}
                  </p>
                  <p className="text-muted-foreground text-xs">
                    {t("auth.mfa.manual_entry_instruction")}
                  </p>
                  <div className="flex items-center gap-2">
                    <code className="bg-muted flex-1 rounded px-3 py-2 font-mono text-sm break-all">
                      {setupData.secret}
                    </code>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleCopySecret}
                    >
                      {secretCopied ? (
                        <CheckCircle className="h-4 w-4 text-green-600" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>
              </div>

              <DialogFooter className="sm:justify-between">
                <Button variant="outline" onClick={handleCloseDialog}>
                  {t("auth.mfa.cancel_setup")}
                </Button>
                <Button onClick={() => setSetupStep("verify")}>
                  {t("auth.continue")}
                </Button>
              </DialogFooter>
            </>
          )}

          {setupStep === "verify" && (
            <>
              <DialogHeader>
                <DialogTitle>{t("auth.mfa.verify_setup_title")}</DialogTitle>
                <DialogDescription>
                  {t("auth.mfa.verify_setup_instruction")}
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4">
                {error && (
                  <div className="flex items-center gap-2 rounded-md bg-red-50 p-3 text-sm text-red-600">
                    <AlertCircle className="h-4 w-4" />
                    {error}
                  </div>
                )}

                <div className="flex justify-center">
                  <InputOTP
                    maxLength={6}
                    value={code}
                    onChange={(value) => setCode(value)}
                    disabled={verifyDeviceMutation.isPending}
                  >
                    <InputOTPGroup>
                      <InputOTPSlot index={0} />
                      <InputOTPSlot index={1} />
                      <InputOTPSlot index={2} />
                      <InputOTPSlot index={3} />
                      <InputOTPSlot index={4} />
                      <InputOTPSlot index={5} />
                    </InputOTPGroup>
                  </InputOTP>
                </div>
              </div>

              <DialogFooter className="sm:justify-between">
                <Button variant="outline" onClick={() => setSetupStep("qr")}>
                  {t("auth.mfa.back_to_login")}
                </Button>
                <Button
                  onClick={handleVerifyCode}
                  disabled={verifyDeviceMutation.isPending || code.length !== 6}
                >
                  {verifyDeviceMutation.isPending
                    ? t("auth.mfa.enabling")
                    : t("auth.mfa.enable_button")}
                </Button>
              </DialogFooter>
            </>
          )}

          {setupStep === "success" && (
            <>
              <div className="flex flex-col items-center space-y-4 py-4">
                <div className="flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
                  <CheckCircle className="h-8 w-8 text-green-600" />
                </div>
                <div className="space-y-2 text-center">
                  <h3 className="text-lg font-semibold">
                    {t("auth.mfa.device_added_title")}
                  </h3>
                  <p className="text-muted-foreground text-sm">
                    {t("auth.mfa.device_added_message")}
                  </p>
                </div>
              </div>

              <DialogFooter>
                <Button onClick={handleCloseDialog} className="w-full">
                  {t("common.close")}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteDeviceId} onOpenChange={handleCloseDeleteDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("auth.mfa.delete_device_title")}</DialogTitle>
            <DialogDescription>
              {t("auth.mfa.delete_device_description")}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {deleteError && (
              <div className="flex items-center gap-2 rounded-md bg-red-50 p-3 text-sm text-red-600">
                <AlertCircle className="h-4 w-4" />
                {deleteError}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="delete-password">
                {t("auth.mfa.delete_password_label")}
              </Label>
              <Input
                id="delete-password"
                type="password"
                value={deletePassword}
                onChange={(e) => setDeletePassword(e.target.value)}
                placeholder={t("auth.mfa.delete_password_placeholder")}
                disabled={removeDeviceMutation.isPending}
              />
            </div>
          </div>

          <DialogFooter className="sm:justify-between">
            <Button variant="outline" onClick={handleCloseDeleteDialog}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={!deletePassword || removeDeviceMutation.isPending}
            >
              {removeDeviceMutation.isPending
                ? t("common.loading")
                : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Reset MFA Dialog */}
      <Dialog open={isResetDialogOpen} onOpenChange={setIsResetDialogOpen}>
        <DialogContent className="sm:max-w-md">
          {resetStep === "password" && (
            <>
              <DialogHeader>
                <DialogTitle>{t("auth.mfa.reset_title")}</DialogTitle>
                <DialogDescription>
                  {t("auth.mfa.reset_password_instruction")}
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4">
                {resetError && (
                  <div className="flex items-center gap-2 rounded-md bg-red-50 p-3 text-sm text-red-600">
                    <AlertCircle className="h-4 w-4" />
                    {resetError}
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="reset-password">
                    {t("auth.mfa.reset_password_label")}
                  </Label>
                  <Input
                    id="reset-password"
                    type="password"
                    value={resetPassword}
                    onChange={(e) => setResetPassword(e.target.value)}
                    placeholder={t("auth.mfa.reset_password_placeholder")}
                    disabled={requestResetMutation.isPending}
                  />
                </div>
              </div>

              <DialogFooter className="sm:justify-between">
                <Button variant="outline" onClick={handleCloseResetDialog}>
                  {t("common.cancel")}
                </Button>
                <Button
                  onClick={handleRequestReset}
                  disabled={!resetPassword || requestResetMutation.isPending}
                >
                  {requestResetMutation.isPending
                    ? t("common.loading")
                    : t("auth.continue")}
                </Button>
              </DialogFooter>
            </>
          )}

          {resetStep === "email_sent" && (
            <>
              <DialogHeader>
                <DialogTitle>{t("auth.mfa.reset_email_sent_title")}</DialogTitle>
                <DialogDescription>
                  {t("auth.mfa.reset_email_sent_description")}
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4">
                {resetError && (
                  <div className="flex items-center gap-2 rounded-md bg-red-50 p-3 text-sm text-red-600">
                    <AlertCircle className="h-4 w-4" />
                    {resetError}
                  </div>
                )}

                <div className="flex justify-center">
                  <InputOTP
                    maxLength={6}
                    value={resetCode}
                    onChange={(value) => setResetCode(value.toUpperCase())}
                    disabled={verifyResetMutation.isPending}
                  >
                    <InputOTPGroup>
                      <InputOTPSlot index={0} />
                      <InputOTPSlot index={1} />
                      <InputOTPSlot index={2} />
                      <InputOTPSlot index={3} />
                      <InputOTPSlot index={4} />
                      <InputOTPSlot index={5} />
                    </InputOTPGroup>
                  </InputOTP>
                </div>
              </div>

              <DialogFooter className="sm:justify-between">
                <Button variant="outline" onClick={handleCloseResetDialog}>
                  {t("common.cancel")}
                </Button>
                <Button
                  onClick={handleVerifyReset}
                  disabled={
                    resetCode.length !== 6 || verifyResetMutation.isPending
                  }
                >
                  {verifyResetMutation.isPending
                    ? t("common.loading")
                    : t("auth.mfa.reset_verify_button")}
                </Button>
              </DialogFooter>
            </>
          )}

          {resetStep === "success" && (
            <>
              <div className="flex flex-col items-center space-y-4 py-4">
                <div className="flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
                  <CheckCircle className="h-8 w-8 text-green-600" />
                </div>
                <div className="space-y-2 text-center">
                  <h3 className="text-lg font-semibold">
                    {t("auth.mfa.reset_success_title")}
                  </h3>
                  <p className="text-muted-foreground text-sm">
                    {t("auth.mfa.reset_success_message")}
                  </p>
                </div>
              </div>

              <DialogFooter>
                <Button onClick={handleCloseResetDialog} className="w-full">
                  {t("common.close")}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
