// Written by go run ./tools/apitypes from the Go types: do not edit.

export interface Checkin {
  unlocked: number;
}

export interface Confusion {
  want: string;
  typed: string;
  n: number;
}

export interface File {
  progress: Progress;
  rounds: Round[];
}

export interface Finger {
  finger: string;
  presses: number;
  mistakes: number;
  rate: number;
  ms: number;
  weak: boolean;
}

export interface FreetypePortion {
  words: string[];
  why?: string[];
}

export interface FreetypePortionRequest {
  recent: string[];
  start: boolean;
  words: number;
  list: number;
  kind: string;
}

export interface FreetypePortions {
  portions: FreetypePortion[];
}

export interface FreetypeResult {
  consistency: number | null;
}

export interface FreetypeStats {
  lists: number[];
  list: number;
  history: History;
  rhythm: Rhythm;
  weakSpots: number;
  standing: Standing;
  owned: number[];
  tired: boolean;
}

export interface History {
  rounds: Round[];
  count: number;
  lengths: string[];
  bests: Record<string, Round>;
}

export interface Imported {
  rounds: number;
}

export interface Key {
  char: string;
  label?: string;
  finger: string;
  home?: boolean;
  width?: number;
  alt?: string;
  modifier?: boolean;
}

export interface KeyStat {
  count: number;
  mistakes: number;
  errEma: number;
  timeEma: number;
  timedCount: number;
  held?: boolean;
  hits?: string;
  slipped?: boolean;
}

export interface Kind {
  kind: string;
  count: number;
  share: number;
}

export interface Layout {
  id: string;
  name: string;
  fingers: Record<string, LayoutFinger>;
  rows: Key[][];
  offsets: number[];
}

export interface LayoutFinger {
  name: string;
  color: string;
}

export interface LearnResult {
  unlocked: number;
  newLetter: string;
  consistency: number | null;
}

export interface LearnStats {
  letters: Letter[];
  unlocked: number;
  mastery: number;
  history: History;
  profile: Profile;
  confusions: Confusion[];
  tired: boolean;
}

export interface Letter {
  char: string;
  count: number;
  level: number;
  avgMs: number;
  mastered: boolean;
  open: boolean;
}

export interface OK {
  ok: boolean;
}

export interface Pair {
  pair: string;
  cost: number;
  count: number;
}

export interface Portion {
  words: string[];
  focus: string;
  weakest: string;
  unlocked: string[];
  locked: string[];
  why?: string[];
}

export interface PortionRequest {
  round: string;
  index: number;
  ack: number;
  start: boolean;
  last: boolean;
  recent: string[];
  retry: string[];
  kind: string;
  level: number;
}

export interface Portions {
  portions: Portion[];
  round: number;
}

export interface Profile {
  presses: number;
  rate: number;
  fingers: Finger[];
  weak: string;
  letters: string[];
  mistakes: number;
  kinds: Kind[];
}

export interface Progress {
  letters: Record<string, KeyStat>;
  unlocked: number;
  lap: number;
}

export interface ReinforceResult {
  newGoal: number;
  consistency: number | null;
}

export interface ReinforceStats {
  letters: Letter[];
  goal: number;
  step: number;
  lastStep: number;
  wordLevels: number[];
  standing: Standing;
  pairs: Pair[];
  owned: number[];
  weakSpots: number;
  history: History;
  profile: Profile;
  tired: boolean;
}

export interface Result {
  id: string;
  samples: Sample[];
  keys: Sample[];
  wpm: number;
  accuracy: number;
  chars: number;
  perSecond: number[];
  length: string;
  kind: string;
  level: number;
  list: number;
}

export interface Rhythm {
  median: number;
  spread: number;
  threshold: number;
  rate: number;
  presses: number;
  pairs: Stumble[];
}

export interface Round {
  id?: string;
  mode: string;
  kind?: string;
  level?: number;
  lap?: number;
  length: string;
  wpm: number;
  accuracy: number;
  chars: number;
  consistency?: number | null;
  unlocked?: number;
  finishedAt: string;
  keys?: Sample[];
}

export interface Sample {
  char: string;
  errors: number;
  ms: number;
  typed?: string;
}

export interface Standing {
  level: number;
  passes: number;
  fails: number;
  moved: number;
  passesUp: number;
  failsDown: number;
}

export interface Step {
  step: number;
  goal: number;
}

export interface Stumble {
  pair: string;
  rate: number;
  count: number;
  stumbles: number;
}
