import { Skeleton } from '@/shared/ui';

/**
 * ScreenSkeleton — the Suspense fallback shown while a lazily-loaded route
 * chunk downloads. Mirrors the Screen layout so there is no jump when the
 * real page mounts.
 */
export function ScreenSkeleton() {
  return (
    <div className="px-5 pb-10 pt-[max(var(--safe-top),1.5rem)]">
      <Skeleton className="h-8 w-1/2" />
      <Skeleton className="mt-3 h-4 w-2/3" />
      <div className="mt-7 space-y-3">
        <Skeleton className="h-24 w-full rounded-card" />
        <Skeleton className="h-24 w-full rounded-card" />
      </div>
    </div>
  );
}
