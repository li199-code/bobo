import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Input,
} from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function SignInPage() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState("");

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    try {
      const response = await fetch("/api/v1/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ username, password }), // 修改：使用 username
      });

      const result = await response.json();

      if (result.status === "ok") {
        localStorage.setItem("token", result.token);
        window.location.href = "/dashboard";
      } else {
        setError("Invalid credentials");
      }
    } catch (err) {
      setError("An error occurred. Please try again.");
      console.error(err);
    }
  };


  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Login</CardTitle>
          <CardDescription>Enter your credentials to sign in.</CardDescription>
        </CardHeader>
        <form onSubmit={handleLogin}>
            <CardContent className="space-y-4">
            {error && <p className="text-red-500 text-sm">{error}</p>}
            <div className="space-y-2">
                <label htmlFor="username">Username</label>
                <Input
                id="username"
                placeholder="admin"
                value={username}
                onChange={(e) => setUsername(e.target.value)} // 修改：更新用户名输入
                required
                />
            </div>
            <div className="space-y-2">
                <label htmlFor="password">Password</label>
                <Input
                id="password"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                />
            </div>
            </CardContent>
          <CardFooter className="flex flex-col space-y-2 mt-4">
            <Button type="submit" className="w-full">Sign In</Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}