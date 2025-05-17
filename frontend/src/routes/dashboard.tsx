import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "@/components/ui/card";

export default function DashboardPage() {
  const [url, setUrl] = useState("");
  const [message, setMessage] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setMessage("");

    try {
      const token = localStorage.getItem("token");
      if (!token) {
        setMessage("未授权，请先登录");
        return;
      }
        
        console.log('====', token)

      const response = await fetch("/api/v1/admin", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Basic ${token}`,
        },
        body: JSON.stringify({ url }),
      });
        

      const result = await response.json();

      if (result.status === "ok") {
        setMessage("添加成功！");
        setUrl(""); // 清空输入框
      } else {
        setMessage("添加失败，请检查 URL 或权限");
      }
    } catch (err) {
      setMessage("请求出错，请重试");
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Dashboard</CardTitle>
          <CardDescription>Add a new blog feed URL to the database.</CardDescription>
        </CardHeader>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-4">
            {message && (
              <p
                className={`text-sm ${
                  message.includes("成功") ? "text-green-500" : "text-red-500"
                }`}
              >
                {message}
              </p>
            )}
            <div className="space-y-2">
              <label htmlFor="url">Blog Feed URL</label>
              <Input
                id="url"
                placeholder="https://www.meizg.cn/feed/"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                required
              />
            </div>
          </CardContent>
          <CardFooter>
            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? "提交中..." : "提交"}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}