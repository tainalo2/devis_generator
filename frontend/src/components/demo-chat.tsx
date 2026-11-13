import { Send } from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"

interface Message {
  id: string
  role: "user" | "assistant"
  content: string
}

/**
 * DemoChat — Interactive demo component showcasing Devis Generator MCP capabilities.
 * Shows conversational interaction with the system for creating invoices and quotes.
 */
export function DemoChat() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "1",
      role: "assistant",
      content:
        "Bonjour ! Je suis l'assistant Devis Generator. Je peux vous aider à créer des devis et factures conformes. Qu'aimeriez-vous faire ?",
    },
  ])
  const [input, setInput] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const handleSend = async () => {
    if (!input.trim()) return

    // Add user message
    const userMessage: Message = {
      id: String(Date.now()),
      role: "user",
      content: input,
    }

    setMessages((prev) => [...prev, userMessage])
    setInput("")
    setIsLoading(true)

    // Simulate assistant response
    setTimeout(() => {
      const assistantMessage: Message = {
        id: String(Date.now() + 1),
        role: "assistant",
        content:
          "Excellente question ! Vous pouvez créer un devis en quelques clics. Chaque document est numéroté automatiquement et conforme à la réglementation française actuelle.",
      }
      setMessages((prev) => [...prev, assistantMessage])
      setIsLoading(false)
    }, 500)
  }

  return (
    <div className="flex flex-col h-96 rounded-lg border border-border bg-card shadow-lg">
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
          >
            <div
              className={`max-w-xs px-4 py-2 rounded-lg ${
                msg.role === "user"
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground"
              }`}
            >
              <p className="text-sm">{msg.content}</p>
            </div>
          </div>
        ))}
        {isLoading && (
          <div className="flex justify-start">
            <div className="bg-muted px-4 py-2 rounded-lg">
              <p className="text-sm text-muted-foreground">Assistant est en train d'écrire...</p>
            </div>
          </div>
        )}
      </div>

      {/* Input area */}
      <div className="border-t border-border p-4">
        <div className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault()
                handleSend()
              }
            }}
            placeholder="Posez une question ou décrivez votre besoin..."
            className="flex-1 px-3 py-2 text-sm rounded-md border border-input bg-background"
            disabled={isLoading}
          />
          <Button
            size="sm"
            onClick={handleSend}
            disabled={!input.trim() || isLoading}
            className="px-3"
          >
            <Send className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
