"use client";

import { useCallback } from "react";
import { useTheme } from "next-themes";
import { MoonIcon, SunIcon } from "lucide-react";

const SYSTEM_THEME = {
  LIGHT: "light",
  DARK: "dark",
} as const;

const USER_THEME = {
  ...SYSTEM_THEME,
  SYSTEM: "system",
} as const;

type SystemTheme = (typeof SYSTEM_THEME)[keyof typeof SYSTEM_THEME];
type UserTheme = (typeof USER_THEME)[keyof typeof USER_THEME];

type ThemeSelectSpec<T> = {
  light: T;
  dark: T;
  system: T;
  default: T;
};

type SystemThemeSelectSpec<T> = {
  light: T;
  dark: T;
};

const getBaseRules = <T,>(light: T, dark: T) => ({
  light,
  dark,
});

const getSystemTheme = (): SystemTheme => {
  if (typeof window !== "undefined") {
    return window.matchMedia(`(prefers-color-scheme: ${SYSTEM_THEME.DARK})`).matches
      ? SYSTEM_THEME.DARK
      : SYSTEM_THEME.LIGHT;
  }
  return SYSTEM_THEME.LIGHT;
};

const selectSystem = <T,>(spec: SystemThemeSelectSpec<T>): T => {
  switch (getSystemTheme()) {
    case SYSTEM_THEME.LIGHT:
      return spec.light;
    case SYSTEM_THEME.DARK:
      return spec.dark;
  }
};

export function useThemeUtils() {
  const { theme, setTheme } = useTheme();

  const select = useCallback(
    <T,>(spec: ThemeSelectSpec<T>): T => {
      switch (theme as UserTheme | undefined) {
        case USER_THEME.LIGHT:
          return spec.light;
        case USER_THEME.DARK:
          return spec.dark;
        case USER_THEME.SYSTEM:
          return spec.system;
        default:
          return spec.default;
      }
    },
    [theme],
  );

  const toggle = useCallback(() => {
    const baseRules = getBaseRules(USER_THEME.DARK, USER_THEME.LIGHT);
    const targetTheme = select({
      ...baseRules,
      system: selectSystem({ ...baseRules }),
      default: USER_THEME.SYSTEM,
    });
    setTheme(targetTheme);
    return targetTheme;
  }, [setTheme, select]);

  const getIcon = (className: string) => {
    const baseRules = getBaseRules(
      <MoonIcon className={className} />,
      <SunIcon className={className} />,
    );
    return select({
      ...baseRules,
      system: selectSystem({ ...baseRules }),
      default: <MoonIcon className={className} />,
    });
  };

  const getAction = () => {
    const baseRules = getBaseRules("切换深色模式", "切换浅色模式");
    return select({
      ...baseRules,
      system: selectSystem({ ...baseRules }),
      default: "切换深色模式",
    });
  };

  return {
    select,
    toggle,
    getIcon,
    getAction,
    selectSystem,
    getSystemTheme,
  };
}
