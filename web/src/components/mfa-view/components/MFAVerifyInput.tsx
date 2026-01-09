import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp";
import { MFA_CODE_LENGTH } from "@/components/mfa-view/helpers/constants";

interface MFAVerifyInputProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  uppercase?: boolean;
}

export function MFAVerifyInput({
  value,
  onChange,
  disabled = false,
  uppercase = false,
}: MFAVerifyInputProps) {
  const handleChange = (newValue: string) => {
    onChange(uppercase ? newValue.toUpperCase() : newValue);
  };

  return (
    <div className="flex justify-center">
      <InputOTP
        maxLength={MFA_CODE_LENGTH}
        value={value}
        onChange={handleChange}
        disabled={disabled}
        autoComplete="one-time-code"
        name="totp"
        id="totp-code"
        aria-label="One-time code"
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
  );
}
