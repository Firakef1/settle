import type { NextConfig } from "next";

// The Go API sends no CORS headers, so the browser can't call it from another
// origin. Proxy /api/v1/* through Next instead: the app calls its own origin.
const apiProxyTarget = process.env.API_PROXY_TARGET || "http://localhost:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/v1/:path*", destination: `${apiProxyTarget}/api/v1/:path*` }];
  },
};

export default nextConfig;
