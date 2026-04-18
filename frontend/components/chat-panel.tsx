"use client";

import { useState, useRef, useEffect } from "react";
import { MessageSquare, Send, Bot, User, FileText, X, Minimize2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { chatWithKnowledge } from "@/lib/api";
import type { ChatResponse, SearchResult } from "@/lib/api";

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  sources?: SearchResult[];
  timestamp: Date;
}

export function ChatPanel() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>("");
  const [isExpanded, setIsExpanded] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages]);

  const handleSendMessage = async () => {
    if (!inputValue.trim()) return;

    const userMessage: Message = {
      id: `msg-${Date.now()}`,
      role: "user",
      content: inputValue,
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInputValue("");
    setIsLoading(true);

    try {
      const response: ChatResponse = await chatWithKnowledge(inputValue, sessionId);

      if (response.session_id && !sessionId) {
        setSessionId(response.session_id);
      }

      const assistantMessage: Message = {
        id: `msg-${Date.now()}-ai`,
        role: "assistant",
        content: response.answer,
        sources: response.sources,
        timestamp: new Date(),
      };

      setMessages((prev) => [...prev, assistantMessage]);
    } catch (error) {
      console.error("Chat error:", error);
      toast.error("Failed to get response from AI");
    } finally {
      setIsLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  if (!isExpanded) {
    return (
      <div className="fixed bottom-6 right-6 z-40">
        <Button
          onClick={() => setIsExpanded(true)}
          size="lg"
          className="rounded-full w-14 h-14 shadow-lg"
        >
          <MessageSquare className="w-6 h-6" />
        </Button>
      </div>
    );
  }

  return (
    <Card className="fixed bottom-6 right-6 w-96 h-[500px] flex flex-col shadow-xl z-40 bg-white">
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-2">
          <Bot className="w-5 h-5 text-blue-600" />
          <h3 className="font-semibold">AI 问答</h3>
        </div>
        <div className="flex gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsExpanded(false)}
            className="h-8 w-8 p-0"
          >
            <Minimize2 className="w-4 h-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setMessages([]);
              setSessionId("");
            }}
            className="h-8 w-8 p-0"
          >
            <X className="w-4 h-4" />
          </Button>
        </div>
      </div>

      <ScrollArea className="flex-1 p-4 overflow-hidden">
        <div className="space-y-4 pr-4">
          {messages.length === 0 && (
            <div className="flex flex-col items-center justify-center h-full text-center text-gray-500 py-8">
              <MessageSquare className="w-12 h-12 mb-2 opacity-50" />
              <p className="text-sm">开始提问，获取知识库中的答案</p>
            </div>
          )}

           {messages.map((message) => (
             <div
               key={message.id}
               className={`flex ${message.role === "user" ? "justify-end" : "justify-start"}`}
             >
               <div
                 className={`flex gap-2 max-w-[70%] ${
                   message.role === "user" ? "flex-row-reverse" : "flex-row"
                 }`}
               >
                <div
                  className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${
                    message.role === "user"
                      ? "bg-blue-600 text-white"
                      : "bg-gray-200 text-gray-700"
                  }`}
                >
                  {message.role === "user" ? (
                    <User className="w-4 h-4" />
                  ) : (
                    <Bot className="w-4 h-4" />
                  )}
                </div>

                <div className="flex flex-col gap-2">
                  <div
                    className={`px-3 py-2 rounded-lg ${
                      message.role === "user"
                        ? "bg-blue-600 text-white"
                        : "bg-gray-100 text-gray-900"
                    }`}
                  >
                    <p className="text-sm whitespace-pre-wrap break-words">
                      {message.content}
                    </p>
                  </div>

                  {message.sources && message.sources.length > 0 && (
                    <div className="space-y-1">
                      {message.sources.map((source, idx) => (
                        <div
                          key={idx}
                          className="bg-blue-50 border border-blue-200 rounded p-2 text-xs cursor-pointer hover:bg-blue-100 transition"
                        >
                          <div className="flex items-start gap-2">
                            <FileText className="w-3 h-3 mt-0.5 flex-shrink-0 text-blue-600" />
                            <div className="flex-1 min-w-0">
                              <p className="font-medium text-blue-900 truncate">
                                {source.title}
                              </p>
                              <p className="text-blue-700 line-clamp-2">
                                {source.snippet}
                              </p>
                              <p className="text-blue-600 mt-1">
                                相关度: {(source.score * 100).toFixed(0)}%
                              </p>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}

          {isLoading && (
            <div className="flex justify-start">
              <div className="flex gap-2">
                <div className="flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center bg-gray-200">
                  <Bot className="w-4 h-4 text-gray-700" />
                </div>
                <div className="bg-gray-100 px-3 py-2 rounded-lg">
                  <div className="flex gap-1">
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" />
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce delay-100" />
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce delay-200" />
                  </div>
                </div>
              </div>
            </div>
          )}

          <div ref={scrollRef} />
        </div>
      </ScrollArea>

      <div className="border-t p-4 flex gap-2">
        <Input
          placeholder="输入问题..."
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isLoading}
          className="flex-1"
        />
        <Button
          onClick={handleSendMessage}
          disabled={isLoading || !inputValue.trim()}
          size="sm"
          className="px-3"
        >
          <Send className="w-4 h-4" />
        </Button>
      </div>
    </Card>
  );
}
