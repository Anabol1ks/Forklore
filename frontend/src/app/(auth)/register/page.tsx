"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { toast } from "sonner";
import axios from "axios";
import { useAuthStore } from "@/store/auth";

export default function RegisterPage() {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const login = useAuthStore((state) => state.login);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const response = await api.post("/auth/register", { username, email, password });
      const { access_token, refresh_token } = response.data.tokens;
      const user = response.data.user;
      login(access_token, refresh_token, user);
      toast.success("Регистрация успешна!");
      router.push("/");
    } catch (error: unknown) {
      if (axios.isAxiosError(error)) {
        toast.error((error.response?.data as { message?: string } | undefined)?.message || "Ошибка регистрации.");
      } else {
        toast.error("Ошибка регистрации.");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex justify-center items-center py-20">
      <Card className="w-full max-w-md">
        <form onSubmit={handleSubmit}>
          <CardHeader>
            <CardTitle className="text-2xl text-center">Регистрация</CardTitle>
            <CardDescription className="text-center">Создайте аккаунт в Forklore</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
             <div className="space-y-2">
              <Label htmlFor="username">Имя пользователя</Label>
              <Input 
                id="username" 
                type="text" 
                placeholder="username" 
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input 
                id="email" 
                type="email" 
                placeholder="name@example.com" 
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Пароль</Label>
              <Input 
                id="password" 
                type="password" 
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
          </CardContent>
          <CardFooter className="flex flex-col space-y-4">
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Загрузка..." : "Зарегистрироваться"}
            </Button>
            <div className="text-sm text-center text-muted-foreground">
              Уже есть аккаунт?{" "}
              <Link href="/login" className="text-primary hover:underline">
                Войти
              </Link>
            </div>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
