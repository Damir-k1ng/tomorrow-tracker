import { lazy, Suspense, type ComponentType } from 'react';
import { ScreenSkeleton } from '@/widgets';

/**
 * Wrap a dynamically-imported page in React.lazy + Suspense.
 *
 * Each call produces its own code-split chunk, so a route's JS is downloaded
 * only when the user navigates to it — and the entire Admin route tree never
 * reaches a non-admin's device.
 */
export function lazyPage(
  importer: () => Promise<{ default: ComponentType }>,
): React.ReactNode {
  const Lazy = lazy(importer);
  return (
    <Suspense fallback={<ScreenSkeleton />}>
      <Lazy />
    </Suspense>
  );
}
