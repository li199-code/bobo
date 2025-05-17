import { useEffect, useState } from "react";

interface BlogPost {
  Title: string;
  Link: string;
  PubDate: string;
}

function App() {
  const [posts, setPosts] = useState<BlogPost[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch("/api/v1/index");
        if (!response.ok) throw new Error("Network response was not ok");

        const result = await response.json();
        setPosts(result.data);
      } catch (error) {
        console.error("Error fetching data:", error);
      }
    };

    fetchData();
  }, []);

  return (
    <div className="flex flex-col items-center justify-center min-h-svh p-4">
      <h1 className="text-2xl font-bold mb-6">Latest Blog Posts</h1>
      <div className="space-y-4 w-full max-w-2xl">
        {posts.length > 0 ? (
          posts.map((post, index) => (
            <div key={index} className="border rounded-lg p-4 shadow-md hover:shadow-lg transition-shadow">
              <a href={post.Link} target="_blank" rel="noopener noreferrer" className="text-blue-600 font-semibold text-lg hover:underline">
                {post.Title}
              </a>
              <p className="text-gray-500 text-sm mt-1">{new Date(post.PubDate).toLocaleDateString()}</p>
            </div>
          ))
        ) : (
          <p>Loading...</p>
        )}
      </div>
    </div>
  );
}

export default App;