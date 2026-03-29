import { type HTMLAttributes, type MouseEvent, forwardRef, useCallback, useEffect } from 'react'

import { cn } from '@/lib/utils'

interface DialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  children: React.ReactNode
}

export function Dialog({ open, onOpenChange, children }: DialogProps) {
  useEffect(() => {
    if (!open) return

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onOpenChange(false)
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [open, onOpenChange])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="fixed inset-0 bg-black/60 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
        data-testid="dialog-overlay"
      />
      <div className="relative z-50 w-full max-w-lg">{children}</div>
    </div>
  )
}

const DialogContent = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, onClick, ...props }, ref) => {
    const handleClick = useCallback(
      (e: MouseEvent<HTMLDivElement>) => {
        e.stopPropagation()
        onClick?.(e)
      },
      [onClick],
    )

    return (
      <div
        ref={ref}
        className={cn(
          'mx-4 rounded-lg border bg-card p-6 shadow-lg sm:mx-0',
          className,
        )}
        onClick={handleClick}
        {...props}
      />
    )
  },
)

DialogContent.displayName = 'DialogContent'

function DialogHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('mb-4 space-y-1.5', className)} {...props} />
}

function DialogTitle({ className, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return <h2 className={cn('text-lg font-semibold leading-none tracking-tight', className)} {...props} />
}

function DialogDescription({ className, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn('text-sm text-muted-foreground', className)} {...props} />
}

function DialogFooter({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('mt-6 flex justify-end gap-2', className)} {...props} />
}

export { DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter }
