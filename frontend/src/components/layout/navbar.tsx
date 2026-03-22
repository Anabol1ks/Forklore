"use client";

import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { BookOpen, LogOut, Plus, UserCircle, Sun, Moon, Search } from "lucide-react";
import { useTheme } from "next-themes";
import { useMounted } from "@/hooks/use-mounted";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useRouter } from "next/navigation";
import { useState, type KeyboardEvent } from "react";

export function Navbar() {
  const { user, isAuthenticated, logout, logoutAll } = useAuthStore();
  const { setTheme, theme } = useTheme();
  const mounted = useMounted();
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState("");

  const handleSearchSubmit = () => {
    const value = searchQuery.trim();
    if (!value) {
      router.push("/search");
      return;
    }
    router.push(`/search?q=${encodeURIComponent(value)}`);
  };

  const handleSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      handleSearchSubmit();
    }
  };

  return (
    <nav className="border-b bg-background sticky top-0 z-50">
      <div className="container mx-auto flex h-16 items-center px-4 justify-between">
        <div className="flex items-center gap-4 md:gap-6 min-w-0">
          <Link href="/" className="flex items-center space-x-2 font-bold text-xl">
            <BookOpen className="h-6 w-6" />
            <span>Forklore</span>
          </Link>
          <div className="hidden md:flex space-x-4 shrink-0">
            <Link href="/" className="text-sm font-medium hover:underline underline-offset-4">Репозитории</Link>
            <Link href="/ranking" className="text-sm font-medium hover:underline underline-offset-4">Рейтинг</Link>
          </div>
          <div className="hidden md:flex items-center w-90 lg:w-105">
            <div className="relative w-full">
              <Search className="h-4 w-4 text-muted-foreground absolute left-3 top-1/2 -translate-y-1/2" />
              <Input
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={handleSearchKeyDown}
                placeholder="Search repositories, documents, files..."
                className="pl-9 pr-20 h-9"
              />
              <Button
                type="button"
                size="sm"
                variant="ghost"
                className="absolute right-1 top-1/2 -translate-y-1/2 h-7"
                onClick={handleSearchSubmit}
              >
                Go
              </Button>
            </div>
          </div>
        </div>
        
        <div className="flex items-center space-x-2 md:space-x-4">
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={() => router.push("/search")}
            title="Поиск"
          >
            <Search className="h-5 w-5" />
          </Button>

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
                <DropdownMenuTrigger className="inline-flex h-7 items-center gap-2 rounded-md px-2.5 text-[0.8rem] font-medium hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50">
                    <UserCircle className="h-5 w-5" />
                    <span>{user.username}</span>
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
