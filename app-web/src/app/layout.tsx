import type { Metadata } from "next";
import { Plus_Jakarta_Sans, Noto_Sans_Thai, Geist_Mono } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "@/lib/auth-context";
import { LangProvider } from "@/lib/lang-context";
import { WorkspaceProvider } from "@/lib/workspace-context";
import { SiteHeader } from "./site-header";

// Bilingual type system: Plus Jakarta Sans carries Latin (a modern SaaS sans),
// Noto Sans Thai carries Thai glyphs. The CSS font stack lists Latin first so
// Latin text renders in Jakarta and Thai text falls back to Noto Sans Thai
// per-glyph. (Replaces the accidental Arial override + Latin-only Geist.)
const jakarta = Plus_Jakarta_Sans({
  variable: "--font-jakarta",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  display: "swap",
});

const notoThai = Noto_Sans_Thai({
  variable: "--font-noto-thai",
  subsets: ["thai", "latin"],
  weight: ["400", "500", "600", "700"],
  display: "swap",
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Prasankit",
  description: "Multi-tenant project management platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${jakarta.variable} ${notoThai.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">
        <LangProvider>
          <AuthProvider>
            <WorkspaceProvider>
              <SiteHeader />
              {children}
            </WorkspaceProvider>
          </AuthProvider>
        </LangProvider>
      </body>
    </html>
  );
}
