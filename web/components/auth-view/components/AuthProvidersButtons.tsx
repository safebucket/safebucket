"use client";

import React, { FC } from "react";

import Image from "next/image";

import { useProvidersData } from "@/components/auth-view/hooks/useProvidersData";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { Button } from "@/components/ui/button";

export const AuthProvidersButtons: FC = () => {
  const { login } = useSessionContext();
  const { providers } = useProvidersData();

  return (
    <div className="mt-4 grid grid-cols-2 gap-4">
      {providers.map((provider) => (
        <Button
          key={provider.id}
          variant="outline"
          onClick={() => login(provider.id)}
        >
          <Image
            width={15}
            height={15}
            alt={`${provider.name} logo`}
            src={`/${provider.id}.svg`}
            onError={(e) => {
              const target = e.target as HTMLImageElement;
              target.src = "/login.svg";
            }}
            className="mr-2 h-4 w-4"
          />
          Continue with {provider.name}
        </Button>
      ))}
    </div>
  );
};
