import { ArrowRight } from 'lucide-react'

import { cn } from '@/lib/utils'

interface DiscoveryPipelineProps {
  candidates: number
  generated: number
  swept: number
  validated: number
  deployed: number
}

interface StageProps {
  label: string
  count: number
}

function Stage({ label, count }: StageProps) {
  const active = count > 0

  return (
    <div
      className={cn(
        'flex flex-col items-center gap-1 rounded-lg border px-4 py-2.5 text-center',
        active
          ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400'
          : 'border-border bg-muted/40 text-muted-foreground',
      )}
    >
      <span className="text-lg font-semibold tabular-nums">{count}</span>
      <span className="font-mono text-[10px] uppercase tracking-[0.16em]">{label}</span>
    </div>
  )
}

export function DiscoveryPipeline({
  candidates,
  generated,
  swept,
  validated,
  deployed,
}: DiscoveryPipelineProps) {
  const stages: StageProps[] = [
    { label: 'Screen', count: candidates },
    { label: 'Generate', count: generated },
    { label: 'Sweep', count: swept },
    { label: 'Validate', count: validated },
    { label: 'Deploy', count: deployed },
  ]

  return (
    <div className="flex flex-wrap items-center gap-2">
      {stages.map((stage, i) => (
        <div key={stage.label} className="flex items-center gap-2">
          <Stage {...stage} />
          {i < stages.length - 1 && (
            <ArrowRight className="size-4 shrink-0 text-muted-foreground" />
          )}
        </div>
      ))}
    </div>
  )
}
