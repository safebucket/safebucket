import type { FC } from "react";

import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface IButtonGroupProps {
  options: Array<any>;
  currentOption: string;
  setCurrentOption: (option: any) => void;
}

export const ButtonGroup: FC<IButtonGroupProps> = ({
  options,
  currentOption,
  setCurrentOption,
}: IButtonGroupProps) => {
  return (
    <div className="bg-muted inline-flex h-10 items-center justify-center rounded-md">
      {options.map((option, i) => (
        <TooltipProvider key={i}>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                key={option.key}
                variant={currentOption == option.key ? "group" : "ghost"}
                size="icon"
                className={
                  currentOption !== option.key ? "text-muted-foreground" : ""
                }
                onClick={() => setCurrentOption(option.key)}
                name="too"
              >
                {option.value}
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>{option.tooltip}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ))}
    </div>
  );
};
