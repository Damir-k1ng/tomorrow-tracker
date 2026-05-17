/**
 * Trigger a browser download for a Blob produced by the backend.
 *
 * Used for backend-generated CSV exports — the frontend only streams the bytes
 * the server sends; it never builds CSV client-side.
 */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}
