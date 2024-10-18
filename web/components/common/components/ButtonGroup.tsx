import * as React from "react";
import { FC } from "react";

import { Button } from "@/components/ui/button";

interface IButtonGroupProps {
  options: any[];
  currentOption: string;
  setCurrentOption: (option: any) => void;
}

export const ButtonGroup: FC<IButtonGroupProps> = ({
  options,
  currentOption,
  setCurrentOption,
}: IButtonGroupProps) => {
  return (
    <div className="inline-flex h-10 items-center justify-center rounded-md bg-muted">
      {options.map((option) => (
        <Button
          key={option.key}
          variant={currentOption == option.key ? "group" : "ghost"}
          size="icon"
          className={
            currentOption !== option.key ? "text-muted-foreground" : ""
          }
          onClick={() => setCurrentOption(option.key)}
        >
          {option.value}
        </Button>
      ))}
    </div>
  );
};
