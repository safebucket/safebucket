import React, { FC, useState } from "react";

import { UserPlus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export interface IAddMembersInputProps {
  onAddEmail: (email: string) => void;
}

export const AddMembersInput: FC<IAddMembersInputProps> = ({
  onAddEmail,
}: IAddMembersInputProps) => {
  const [email, setEmail] = useState<string>("");

  const handleAddEmail = (e: React.MouseEvent) => {
    e.preventDefault();
    onAddEmail(email);
    setEmail("");
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      onAddEmail(email);
      setEmail("");
    }
  };

  return (
    <div className="grid grid-cols-12 items-center gap-4">
      <Label htmlFor="share_with_email" className="col-span-2">
        Share with
      </Label>
      <div className="col-span-10 flex space-x-2">
        <Input
          id="share_with_email"
          type="email"
          placeholder="Enter email address"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          onKeyDown={handleKeyPress}
        />
        <Button
          variant="secondary"
          className="shrink-0"
          onClick={handleAddEmail}
          disabled={!email.trim()}
        >
          <UserPlus className="h-4 w-4" />
          Add
        </Button>
      </div>
    </div>
  );
};
