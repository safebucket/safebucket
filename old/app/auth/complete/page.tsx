"use client";

import React, { useEffect } from "react";

import { useRouter } from "next/navigation";

import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { LoadingView } from "@/components/common/components/LoadingView";

export default function Login() {
  const router = useRouter();
  const { status } = useSessionContext();

  useEffect(() => {
    if (status == "authenticated") {
      router.push("/");
    }
  }, [router, status]);

  return <LoadingView />;
}
