// Material Symbols Outlined is loaded in app/layout.tsx; the Figma icons are these glyphs.
// Size is set inline because Google's stylesheet fixes font-size at 24px and,
// being unlayered, beats Tailwind's text-[..] utilities.
export function Icon({ name, size = 20, className = '' }: { name: string; size?: number; className?: string }) {
  return (
    <span
      aria-hidden
      className={`material-symbols-outlined shrink-0 select-none leading-none ${className}`}
      style={{ fontSize: size, width: size, height: size }}
    >
      {name}
    </span>
  );
}
