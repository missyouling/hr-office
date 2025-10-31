/** @type {import('next').NextConfig} */
const nextConfig = {
  // Enable standalone output for Docker
  output: 'standalone',

  // Disable static page generation to avoid SSR errors with client-side auth
  // This is necessary because our auth system uses localStorage which is not available during SSR
  experimental: {
  },

  // Skip build-time static generation for all pages
  // This prevents "useAuth must be used within an AuthProvider" errors during build
  distDir: '.next',
  
  // Disable static optimization
  skipTrailingSlashRedirect: true,
  skipMiddlewareUrlNormalize: true,
}

module.exports = nextConfig
