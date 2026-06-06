import type { Metadata } from "next";
import {
  Fraunces,
  Geist,
  Geist_Mono,
  Noto_Sans_SC,
  Noto_Serif_SC,
} from "next/font/google";
import "./globals.css";

const fraunces = Fraunces({
  variable: "--font-display",
  subsets: ["latin"],
  axes: ["opsz", "SOFT", "WONK"],
});

const geist = Geist({
  variable: "--font-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
});

// 中文字形：Fraunces / Geist 只含拉丁字形，中文会退化成系统宋体。
// 这两款 Noto SC 作为 CJK 兜底接进 font-display / font-sans / font-mono 的
// fallback 链（见 globals.css），让中文标题与正文也用上设计字体。CJK 字库体积
// 大，preload:false 避免整包预加载，按需加载。
const notoSerifSC = Noto_Serif_SC({
  variable: "--font-display-cjk",
  weight: ["400", "600"],
  subsets: ["latin"],
  preload: false,
});

const notoSansSC = Noto_Sans_SC({
  variable: "--font-sans-cjk",
  weight: ["400", "500", "700"],
  subsets: ["latin"],
  preload: false,
});

export const metadata: Metadata = {
  title: "Ashley · 短剧生产工作台",
  description:
    "一个 AI 生产工作台:把一句需求转化为面向美国市场的 Ashley 家具品牌竖屏短剧制作方案。",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="zh-CN"
      className={`${fraunces.variable} ${geist.variable} ${geistMono.variable} ${notoSerifSC.variable} ${notoSansSC.variable} h-full antialiased`}
    >
      <body className="min-h-full">{children}</body>
    </html>
  );
}
