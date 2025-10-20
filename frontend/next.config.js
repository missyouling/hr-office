/** @type {import('next').NextConfig} */
const nextConfig = {
  // Enable standalone output for Docker
  output: 'standalone',

  // Allow cross-origin access for development
  experimental: {
    allowedDevOrigins: ['8.211.154.165']
  }
}

module.exports = nextConfig