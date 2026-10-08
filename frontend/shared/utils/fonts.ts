// Font families come from next/font variables set in app/layout.tsx.
export const fontHeading = 'font-[family-name:var(--font-plus-jakarta-sans)]';
export const fontBody = 'font-[family-name:var(--font-inter)]';

export function initials(name: string | undefined | null): string {
  const parts = (name ?? '').trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  return (parts[0][0] + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase();
}
