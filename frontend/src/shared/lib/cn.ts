import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * Merge class names: clsx resolves conditionals, tailwind-merge dedupes
 * conflicting Tailwind utilities (last one wins). The single class-name helper
 * used across the app.
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
