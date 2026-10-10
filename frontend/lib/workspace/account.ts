/** Initials from the signed-in user's name, never illustrative account data. */
export function accountInitials(name?: string | null): string {
  const words = name?.trim().split(/\s+/).filter(Boolean) ?? [];
  const first = words[0];
  if (!first) return "?";
  return [first, ...(words.length > 1 ? [words[words.length - 1] ?? first] : [])]
    .map(word => Array.from(word)[0])
    .join("")
    .toLocaleUpperCase();
}
