import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Customer dashboard',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="size-full antialiased">
      <body className="relative size-full bg-gradient-to-br from-slate-100 to-green-100">
        {children}
      </body>
    </html>
  );
}
