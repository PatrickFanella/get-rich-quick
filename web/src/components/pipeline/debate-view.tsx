import { useMemo } from 'react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import type { AgentDecision, AgentRole } from '@/lib/api/types';
import { cn } from '@/lib/utils';

const debateLabels: Record<string, string> = {
  bull_researcher: 'Bull',
  bear_researcher: 'Bear',
  aggressive_analyst: 'Aggressive',
  conservative_analyst: 'Conservative',
  neutral_analyst: 'Neutral',
  aggressive_risk: 'Aggressive',
  conservative_risk: 'Conservative',
  neutral_risk: 'Neutral',
};

const roleColors: Record<string, string> = {
  bull_researcher: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400',
  bear_researcher: 'bg-red-500/15 text-red-700 dark:text-red-400',
  aggressive_analyst: 'bg-orange-500/15 text-orange-700 dark:text-orange-400',
  conservative_analyst: 'bg-blue-500/15 text-blue-700 dark:text-blue-400',
  neutral_analyst: 'bg-zinc-500/15 text-zinc-700 dark:text-zinc-400',
  aggressive_risk: 'bg-orange-500/15 text-orange-700 dark:text-orange-400',
  conservative_risk: 'bg-blue-500/15 text-blue-700 dark:text-blue-400',
  neutral_risk: 'bg-zinc-500/15 text-zinc-700 dark:text-zinc-400',
};

/** True when the role should render on the right side of the chat. */
function isRightAligned(role: AgentRole, roles: AgentRole[]): boolean {
  if (roles.length <= 1) return false;
  return roles.indexOf(role) % 2 !== 0;
}

interface DebateViewProps {
  title: string;
  roles: AgentRole[];
  decisions: AgentDecision[];
  onSelectDecision: (decision: AgentDecision) => void;
  isCompleted?: boolean;
}

export function DebateView({
  title,
  roles,
  decisions,
  onSelectDecision,
  isCompleted = false,
}: DebateViewProps) {
  /** Flat, chronologically-ordered list of debate turns across all rounds. */
  const turns = useMemo(() => {
    return decisions
      .filter((d) => roles.includes(d.agent_role))
      .sort((a, b) => {
        const ra = a.round_number ?? 1;
        const rb = b.round_number ?? 1;
        if (ra !== rb) return ra - rb;
        return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
      });
  }, [decisions, roles]);

  const hasMultipleRounds = turns.some((t) => (t.round_number ?? 1) > 1);

  return (
    <div data-testid="debate-view">
      <h3 className="mb-3 text-sm font-semibold text-muted-foreground">{title}</h3>

      {turns.length === 0 ? (
        <Card>
          <CardContent className="py-8">
            <p className="text-center text-sm text-muted-foreground">
              {isCompleted ? 'No debate recorded for this run.' : 'Waiting for debate to begin…'}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {turns.map((entry, idx) => {
            const right = isRightAligned(entry.agent_role, roles);
            const prevRound = idx > 0 ? (turns[idx - 1].round_number ?? 1) : 0;
            const thisRound = entry.round_number ?? 1;
            const showRoundSep = hasMultipleRounds && thisRound !== prevRound;

            return (
              <div key={entry.id}>
                {showRoundSep && (
                  <div
                    className="my-4 flex items-center gap-2 text-xs text-muted-foreground"
                    data-testid="debate-round-indicator"
                  >
                    <div className="h-px flex-1 bg-border" />
                    <span>Round {thisRound}</span>
                    <div className="h-px flex-1 bg-border" />
                  </div>
                )}

                <div className={cn('flex', right ? 'justify-end' : 'justify-start')}>
                  <button
                    type="button"
                    className={cn(
                      'max-w-[85%] cursor-pointer rounded-xl px-4 py-3 text-left transition-shadow hover:shadow-md',
                      'border bg-card',
                      right ? 'rounded-tr-sm' : 'rounded-tl-sm',
                    )}
                    onClick={() => onSelectDecision(entry)}
                    data-testid={`debate-card-${entry.agent_role}`}
                  >
                    <div className="mb-1.5 flex items-center gap-2">
                      <Badge
                        variant="secondary"
                        className={cn(
                          'text-[10px] font-semibold',
                          roleColors[entry.agent_role] ?? '',
                        )}
                      >
                        {debateLabels[entry.agent_role] ?? entry.agent_role}
                      </Badge>
                      {entry.llm_model && (
                        <span className="text-[10px] text-muted-foreground">{entry.llm_model}</span>
                      )}
                      {entry.latency_ms != null && (
                        <span className="text-[10px] text-muted-foreground">
                          {entry.latency_ms}ms
                        </span>
                      )}
                    </div>
                    <p className="whitespace-pre-wrap text-xs leading-relaxed text-foreground">
                      {entry.output_text}
                    </p>
                    <div className="mt-1.5 text-[10px] text-muted-foreground">
                      {new Date(entry.created_at).toLocaleTimeString()}
                    </div>
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
