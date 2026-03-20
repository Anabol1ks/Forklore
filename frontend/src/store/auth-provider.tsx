"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/store/auth";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const { initialize } = useAuthStore();

  useEffect(() => {
    void initialize();
  }, [initialize]);

  return <>{children}</>;
}
