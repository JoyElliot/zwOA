import react from '@vitejs/plugin-react';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const desktop = mode === 'desktop';
  const { ZWOA_DEV_API_TARGET } = loadEnv(mode, '.', 'ZWOA_');

  return {
    plugins: [react()],
    base: desktop ? './' : '/',
    build: {
      outDir: desktop ? 'dist-desktop' : 'dist',
      rolldownOptions: {
        output: {
          codeSplitting: {
            groups: [{
              name: 'react-vendor',
              test: /[\\/]node_modules[\\/](?:react|react-dom|react-router|scheduler)[\\/]/,
            }],
          },
        },
      },
    },
    server: {
      host: '127.0.0.1',
      port: 5273,
      strictPort: true,
      proxy: {
        '/api': { target: ZWOA_DEV_API_TARGET || 'http://127.0.0.1:8720', changeOrigin: false },
      },
    },
  };
});
