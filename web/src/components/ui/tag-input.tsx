import * as React from "react"
import { X } from "lucide-react"
import { Badge, badgeVariants } from "./badge"
import { cn } from "@/lib/utils"
import { type VariantProps } from "class-variance-authority"

export interface TagInputProps {
  value?: string[]
  onChange?: (tags: string[]) => void
  placeholder?: string
  disabled?: boolean
  className?: string
  inputClassName?: string
  badgeVariant?: VariantProps<typeof badgeVariants>["variant"]
  id?: string
  ariaLabel?: string
  allowDuplicates?: boolean
}

export function TagInput({
  value = [],
  onChange,
  placeholder = "Add...",
  disabled = false,
  className,
  inputClassName,
  badgeVariant = "secondary",
  id,
  ariaLabel,
  allowDuplicates = false,
}: TagInputProps) {
  const [inputValue, setInputValue] = React.useState("")
  const inputRef = React.useRef<HTMLInputElement>(null)

  const addTags = React.useCallback(
    (tokens: string[]) => {
      if (!onChange) return
      const currentList = value || []
      const seen = new Set(currentList.map((t) => t.toLowerCase()))
      const newItems: string[] = []

      for (const token of tokens) {
        const trimmed = token.trim()
        if (!trimmed) continue
        if (allowDuplicates || !seen.has(trimmed.toLowerCase())) {
          seen.add(trimmed.toLowerCase())
          newItems.push(trimmed)
        }
      }

      if (newItems.length > 0) {
        onChange([...currentList, ...newItems])
      }
    },
    [value, onChange, allowDuplicates]
  )

  const removeTag = React.useCallback(
    (index: number) => {
      if (!onChange || disabled) return
      const currentList = value || []
      onChange(currentList.filter((_, i) => i !== index))
    },
    [value, onChange, disabled]
  )

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value
    if (val.includes(",")) {
      const parts = val.split(",")
      const toAdd = parts.slice(0, -1)
      addTags(toAdd)
      setInputValue(parts[parts.length - 1])
    } else {
      setInputValue(val)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault()
      if (inputValue.trim()) {
        addTags([inputValue])
        setInputValue("")
      }
    } else if (e.key === "Backspace" && !inputValue && (value || []).length > 0) {
      e.preventDefault()
      removeTag((value || []).length - 1)
    }
  }

  const handleBlur = () => {
    if (inputValue.trim()) {
      addTags([inputValue])
      setInputValue("")
    }
  }

  return (
    <div
      data-slot="tag-input"
      onClick={() => inputRef.current?.focus()}
      className={cn(
        "flex min-h-9 w-full flex-wrap items-center gap-1.5 rounded-lg border border-input bg-transparent px-2.5 py-1.5 text-xs transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30",
        className
      )}
    >
      {(value || []).map((tag, idx) => (
        <Badge
          key={`${tag}-${idx}`}
          variant={badgeVariant}
          className="gap-1 py-0.5 px-2 text-xs font-normal max-w-full"
        >
          <span className="truncate">{tag}</span>
          {!disabled && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                removeTag(idx)
              }}
              className="rounded-full hover:bg-foreground/20 p-0.5 cursor-pointer ml-0.5"
              title={`Remove ${tag}`}
              aria-label={`Remove ${tag}`}
            >
              <X className="size-3" />
            </button>
          )}
        </Badge>
      ))}
      <input
        ref={inputRef}
        id={id}
        type="text"
        value={inputValue}
        disabled={disabled}
        placeholder={(value || []).length === 0 ? placeholder : ""}
        aria-label={ariaLabel || placeholder || "Add tag"}
        onChange={handleInputChange}
        onKeyDown={handleKeyDown}
        onBlur={handleBlur}
        className={cn(
          "flex-1 min-w-[80px] bg-transparent text-xs outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed",
          inputClassName
        )}
      />
    </div>
  )
}
