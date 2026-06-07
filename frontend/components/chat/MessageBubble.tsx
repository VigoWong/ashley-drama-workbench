// frontend/components/chat/MessageBubble.tsx
"use client"

// MessageBubble renders a user or assistant text turn. User bubbles sit right
// (ember), assistant left (neutral).
export default function MessageBubble({ role, text }: { role: "user" | "assistant"; text: string }) {
  if (role === "user") {
    return (
      <div className="my-2 ml-10 rounded-2xl rounded-br-sm bg-ember-500/20 px-3.5 py-2 font-sans text-sm leading-relaxed text-bone-50">
        {text}
      </div>
    )
  }
  return (
    <div className="my-2 mr-10 rounded-2xl rounded-bl-sm bg-ink-800/70 px-3.5 py-2 font-sans text-sm leading-relaxed text-bone-100 whitespace-pre-wrap">
      {text}
    </div>
  )
}
