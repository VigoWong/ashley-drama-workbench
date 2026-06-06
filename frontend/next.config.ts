import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Emit a self-contained server (.next/standalone/server.js) so the Docker
  // runtime image stays small — see frontend/Dockerfile.
  output: "standalone",
};

export default nextConfig;
