export interface ChallengeValidationFormData {
  code: string;
}

export interface ChallengeValidationResponse {
  access_token: string;
  refresh_token: string;
}

export interface ChallengeValidationFormProps {
  onSubmit?: (code: string) => void;
  invitationId?: string;
  challengeId?: string;
}
