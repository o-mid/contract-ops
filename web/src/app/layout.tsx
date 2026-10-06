import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";

const sans = Inter({
  subsets: ["latin"],
  variable: "--font-sans",
  display: "swap"
});

const mono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-mono",
  display: "swap"
});

export const metadata: Metadata = {
  metadataBase: new URL(
    process.env.NEXT_PUBLIC_SITE_URL ?? "https://web-production-a614a.up.railway.app"
  ),
  title: "Contract Ops — integrations control plane",
  description:
    "Open-source integrations control plane with a live activity console, sealed credentials, and sync jobs.",
  openGraph: {
    title: "Contract Ops",
    description: "Integrations control plane with live SSE activity feed",
    images: [{ url: "/og.svg", width: 1200, height: 630 }]
  }
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className={`${sans.variable} ${mono.variable} font-sans antialiased`}>{children}</body>
    </html>
  );
}
