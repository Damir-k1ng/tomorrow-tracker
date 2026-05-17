import { useEffect, useRef } from 'react';
import { bootstrapAuth } from '@/features/auth';
import { useAuthStore } from '@/store';
import { AuthError } from './AuthError';
import { SplashScreen } from './SplashScreen';

/**
 * AuthBootstrap — gates the entire app behind auth resolution.
 *
 *   idle / loading → SplashScreen
 *   error          → AuthError (retry re-runs bootstrap)
 *   authenticated  → children (the role-aware router)
 *
 * `children` — i.e. the route tree — only mounts once the role is known, so
 * the User and Admin apps never flash before role resolution.
 */
export function AuthBootstrap({ children }: { children: React.ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    void bootstrapAuth();
  }, []);

  if (status === 'authenticated') return <>{children}</>;
  if (status === 'error') return <AuthError onRetry={() => void bootstrapAuth()} />;
  return <SplashScreen />;
}
