import { useLayoutEffect, useState } from 'react';

type ColorTheme = 'light' | 'dark';

const storageKey = 'zwoa-color-theme';

function readTheme(): ColorTheme {
  const initial = document.documentElement.dataset.theme;
  if (initial === 'light' || initial === 'dark') return initial;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function useColorTheme() {
  const [colorTheme, setColorTheme] = useState(readTheme);

  useLayoutEffect(() => {
    document.documentElement.dataset.theme = colorTheme;
    document.querySelector('meta[name="theme-color"]')?.setAttribute(
      'content',
      colorTheme === 'dark' ? '#16181e' : '#fdfdfe',
    );
  }, [colorTheme]);

  function toggleTheme() {
    const next = colorTheme === 'light' ? 'dark' : 'light';
    setColorTheme(next);
    try {
      localStorage.setItem(storageKey, next);
    } catch {
      // 存储受限时只在当前窗口应用所选配色。
    }
  }

  return { colorTheme, toggleTheme };
}
