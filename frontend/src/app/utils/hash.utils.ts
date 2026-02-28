export function hashQueryId(input: string): string {
    let hash = 0;

    for (let i = 0; i < input.length; i++) {
        hash = (hash << 5) - hash + input.charCodeAt(i);
        hash |= 0; // force 32-bit integer
    }

    // Returns the absolute decimal representation (0-9 only)
    return Math.abs(hash).toString();
}