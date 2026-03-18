"use client";

import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { BookOpen, LogOut, Plus, UserCircle, Sun, Moon } from "lucide-react";
import { useTheme } from "next-themes";
import { useMounted } from "@/hooks/use-mounted";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useRouter } from "next/navigation";

export function Navbar() {
  const { user, isAuthenticated, logout, logoutAll } = useAuthStore();
  const { setTheme, theme } = useTheme();
  const mounted = useMounted();
  const router = useRouter();

  return (
    <nav className="border-b bg-background sticky top-0 z-50">
      <div className="container mx-auto flex h-16 items-center px-4 justify-between">
        <div className="flex items-center space-x-6">
          <Link href="/" className="flex items-center space-x-2 font-bold text-xl">
            <BookOpen className="h-6 w-6" />
            <span>Forklore</span>
          </Link>
          <div className="hidden md:flex space-x-4">
            <Link href="/" className="text-sm font-medium hover:underline underline-offset-4">Репозитории</Link>
          </div>
        </div>
        
        <div className="flex items-center space-x-4">
          {mounted && (
            <Button variant="ghost" size="icon" onClick={() => setTheme(theme === "dark" ? "light" : "dark")}>
              {theme === "dark" ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
            </Button>
          )}

          {isAuthenticated && user ? (
            <>
              <Link href="/repo/create" title="Создать репозиторий">
                <Button variant="ghost" size="icon">
                  <Plus className="h-5 w-5" />
                </Button>
              </Link>
              <DropdownMenu>
                <DropdownMenuTrigger>
                  <Button variant="ghost" size="sm" className="space-x-2">
                    <UserCircle className="h-5 w-5" />
                    <span>{user.username}</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => router.push(`/user/${user.username}`)}>Профиль</DropdownMenuItem>
                  <DropdownMenuItem onClick={() => void logout()}>
                    <LogOut className="mr-2 h-4 w-4" /> Выйти
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => void logoutAll()}>
                    <LogOut className="mr-2 h-4 w-4" /> Выйти со всех устройств
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          ) : (
            <>
              <Link href="/login">
                <Button variant="ghost">Войти</Button>
              </Link>
              <Link href="/register">
                <Button>Регистрация</Button>
              </Link>
            </>
          )}
        </div>
      </div>
    </nav>
  );
}
