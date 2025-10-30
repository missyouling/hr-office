"use client";

import { Toaster } from "@/components/ui/sonner";
import { AuthProvider } from "@/lib/supabase/auth-context";

export function AppProviders({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      {children}
      <Toaster position="top-right" />
    </AuthProvider>
  );
}

