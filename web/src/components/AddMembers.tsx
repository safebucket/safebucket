import { useState } from "react";

import type { Invites } from "@/types/bucket";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface AddMembersProps {
  shareWith: Array<Invites>;
  onShareWithChange: (shareWith: Array<Invites>) => void;
  currentUserEmail?: string;
  currentUserName?: string;
}

export function AddMembers({
  shareWith,
  onShareWithChange,
  currentUserEmail,
  currentUserName,
}: AddMembersProps) {
  const [email, setEmail] = useState("");
  const [group, setGroup] = useState("viewer");

  const addMember = () => {
    if (email && !shareWith.find((member) => member.email === email)) {
      onShareWithChange([...shareWith, { email, group }]);
      setEmail("");
      setGroup("viewer");
    }
  };

  const removeMember = (emailToRemove: string) => {
    onShareWithChange(
      shareWith.filter((member) => member.email !== emailToRemove),
    );
  };

  return (
    <div className="space-y-4">
      <div>
        <Label>Share with people</Label>
        <div className="mt-2 flex gap-2">
          <Input
            placeholder="Enter email address"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onKeyPress={(e) => e.key === "Enter" && addMember()}
          />
          <select
            value={group}
            onChange={(e) => setGroup(e.target.value)}
            className="rounded-md border px-3 py-2"
          >
            <option value="viewer">Viewer</option>
            <option value="contributor">Contributor</option>
            <option value="owner">Owner</option>
          </select>
          <Button type="button" onClick={addMember}>
            Add
          </Button>
        </div>
      </div>

      {shareWith.length > 0 && (
        <div>
          <Label>People with access:</Label>
          <div className="mt-2 space-y-2">
            {currentUserEmail && (
              <div className="bg-muted flex items-center justify-between rounded p-2">
                <span>{currentUserName || currentUserEmail} (You)</span>
                <span className="text-muted-foreground text-sm">Owner</span>
              </div>
            )}
            {shareWith.map((member) => (
              <div
                key={member.email}
                className="flex items-center justify-between rounded border p-2"
              >
                <span>{member.email}</span>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground text-sm capitalize">
                    {member.group}
                  </span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => removeMember(member.email)}
                  >
                    Remove
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
