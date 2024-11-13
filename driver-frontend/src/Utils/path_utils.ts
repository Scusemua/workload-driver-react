/**
 * This function accepts a path of the form "x/y/z" and returns a path "BASE_URL/x/y/z", where "BASE_URL"
 * is a string retrieved from the PUBLIC_PATH environment variable.
 *
 * This ensures that fetch requests targeting the backend server are sent to the correct path, as this can vary
 * depending on how exactly the server is deployed (e.g., if we're deployed in a Docker Swarm cluster behind a
 * Traefik reverse proxy).
 *
 * @param path The base with no "base path" prefix.
 */
export function GetPathForFetch(path: string): string {
    const basePath: string = process.env.PUBLIC_PATH || '/';
    return JoinPaths(basePath, path);
}

/**
 * Concatenate one or more paths together using forward slashes.
 *
 * IMPORTANT: Do not pass "ws://" or "http://", as the double slashes
 * in those will be reduced to a single forward slash.
 *
 * @param paths the paths to concatenate together.
 */
export function JoinPaths(...paths: string[]): string {
    return paths
        .map((path) => path.trim().replace(/\\/g, '/')) // Normalize all to forward slashes
        .filter(Boolean) // Remove empty strings
        .join('/')
        .replace(/\/+/g, '/'); // Replace multiple slashes with a single one
}
