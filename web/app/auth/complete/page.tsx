"use client";

import React, { useEffect } from "react";

import { useRouter } from "next/navigation";

import { useSession } from "@/app/auth/hooks/useSession";

import { Loading } from "@/components/loading";

export default function Login() {
  const router = useRouter();
  const { status } = useSession();

  useEffect(() => {
    if (status == "authenticated") {
      router.push("/");
    }
  }, [router, status]);

  return <Loading />;
}
