"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Activity, ArrowDown, ArrowLeft, ArrowRight, ArrowUp, Copy, Crown, Gem, Keyboard, Radio, Shield, Swords, Users } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";

type Player = { id: string; name: string; x: number; y: number; score: number; ready: boolean };
type Crystal = { id: string; x: number; y: number; value: number };
type Snapshot = { roomCode: string; status: string; players: Record<string, Player>; crystals: Crystal[]; remainingMs: number };

const demoCrystals: Crystal[] = [
  { id: "c1", x: 18, y: 22, value: 10 }, { id: "c2", x: 72, y: 18, value: 10 },
  { id: "c3", x: 45, y: 38, value: 10 }, { id: "c4", x: 84, y: 64, value: 10 },
  { id: "c5", x: 25, y: 77, value: 10 }, { id: "c6", x: 58, y: 82, value: 10 },
];

const levelDuration = (level: number) => Math.max(30, 60 - (level - 1) * 5) * 1000;
const crystalsForLevel = (level: number): Crystal[] => {
  const positions = [...demoCrystals];
  for (let index = 0; index < Math.min(level - 1, 6); index++) {
    positions.push({ id: `level-${level}-${index}`, x: 12 + ((index * 31 + level * 17) % 76), y: 14 + ((index * 23 + level * 11) % 70), value: 10 });
  }
  return positions;
};

export default function Home() {
  const [name, setName] = useState("Chamindu");
  const [roomCode, setRoomCode] = useState("");
  const [apiUrl, setApiUrl] = useState("http://localhost:8090");
  const [screen, setScreen] = useState<"start" | "lobby" | "game">("start");
  const [status, setStatus] = useState("Offline demo available");
  const [playerId, setPlayerId] = useState("demo-player");
  const [players, setPlayers] = useState<Record<string, Player>>({});
  const [crystals, setCrystals] = useState(demoCrystals);
  const [remaining, setRemaining] = useState(60_000);
  const [demo, setDemo] = useState(false);
  const [level, setLevel] = useState(1);
  const [matchEnded, setMatchEnded] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);
  const sequence = useRef(0);
  const deadline = useRef(0);
  const touchDirection = useRef([0, 0]);
  const playersRef = useRef(players);
  const crystalsRef = useRef(crystals);
  playersRef.current = players;
  crystalsRef.current = crystals;
  const currentPlayer = players[playerId];

  const applySnapshot = useCallback((snapshot: Snapshot) => {
    setPlayers(snapshot.players);
    setCrystals(snapshot.crystals ?? []);
    setRemaining(snapshot.remainingMs);
    if (snapshot.status === "playing") { setMatchEnded(false); setScreen("game"); }
    if (snapshot.status === "finished" || (snapshot.status === "playing" && snapshot.remainingMs <= 0)) setMatchEnded(true);
  }, []);

  const connectSocket = useCallback((code: string, id: string, displayName: string) => {
    socketRef.current?.close();
    setDemo(false); setMatchEnded(false); sequence.current = 0;
    const wsBase = apiUrl.replace(/^http/, "ws");
    const query = new URLSearchParams({ playerId: id, name: displayName });
    const socket = new WebSocket(`${wsBase}/api/v1/rooms/${code}/connect?${query}`);
    socketRef.current = socket;
    socket.onopen = () => { setStatus("Connected to game server"); setScreen("lobby"); };
    socket.onmessage = event => {
      const message = JSON.parse(event.data);
      if (message.type === "snapshot") applySnapshot(message.data);
      if (message.type === "player_joined") setPlayers(old => ({ ...old, [message.data.id]: message.data }));
      if (message.type === "player_left") setPlayers(old => { const next = { ...old }; delete next[message.data.playerId]; return next; });
      if (message.type === "match_finished") {
        setPlayers(old => Object.fromEntries(Object.entries(old).map(([id, player]) => [id, { ...player, score: message.data.scores[id] ?? player.score }])));
        setRemaining(0); setMatchEnded(true); setStatus("Match finished");
      }
      if (message.type === "error") setStatus(String(message.data));
    };
    socket.onerror = () => setStatus("Could not reach backend — check Docker and port 8090");
    socket.onclose = () => { if (socketRef.current === socket) { setMatchEnded(true); setStatus("Disconnected — return to menu and reconnect"); } };
  }, [apiUrl, applySnapshot]);

  const preparePlayer = async () => {
    const response = await fetch(`${apiUrl}/api/v1/players`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ displayName: name }) });
    if (!response.ok) throw new Error("Could not create player");
    return response.json() as Promise<{ playerId: string }>;
  };

  const createRoom = async () => {
    try {
      setStatus("Creating room...");
      const player = await preparePlayer();
      const response = await fetch(`${apiUrl}/api/v1/rooms`, { method: "POST" });
      if (!response.ok) throw new Error("Could not create room");
      const room = await response.json() as { roomCode: string };
      setPlayerId(player.playerId); setRoomCode(room.roomCode); connectSocket(room.roomCode, player.playerId, name);
    } catch (error) { setStatus(error instanceof Error ? error.message : "Connection failed"); }
  };

  const joinRoom = async () => {
    try {
      setStatus("Joining room...");
      const player = await preparePlayer();
      setPlayerId(player.playerId); connectSocket(roomCode.toUpperCase(), player.playerId, name);
    } catch (error) { setStatus(error instanceof Error ? error.message : "Connection failed"); }
  };

  const startDemo = () => {
    const previous = socketRef.current; socketRef.current = null; previous?.close();
    deadline.current = Date.now() + levelDuration(1);
    const id = "demo-player";
    setDemo(true); setLevel(1); setMatchEnded(false); setPlayerId(id); setRoomCode("DEMO42"); setRemaining(levelDuration(1)); setCrystals(crystalsForLevel(1));
    setPlayers({
      [id]: { id, name: name || "Player", x: 50, y: 55, score: 0, ready: true },
      rival: { id: "rival", name: "Nyx", x: 68, y: 42, score: 20, ready: true },
    });
    setStatus("Demo simulation"); setScreen("game");
  };

  const send = useCallback((message: object) => {
    if (socketRef.current?.readyState === WebSocket.OPEN) socketRef.current.send(JSON.stringify(message));
  }, []);

  const movePlayer = useCallback((dx: number, dy: number) => {
    if (matchEnded || (demo && Date.now() >= deadline.current)) return;
    if (!demo) { send({ type: "input", dx, dy, sequence: ++sequence.current }); return; }
    if (!dx && !dy) return;
      const old = playersRef.current;
      const player = old[playerId]; if (!player) return;
      const length = Math.hypot(dx, dy) || 1;
      const moved = { ...player, x: Math.max(2, Math.min(98, player.x + dx / length)), y: Math.max(3, Math.min(97, player.y + dy / length)) };
      const nextCrystals = crystalsRef.current.map(crystal => {
        if (Math.hypot(moved.x - crystal.x, moved.y - crystal.y) > 4) return crystal;
        moved.score += crystal.value;
        return { ...crystal, x: 8 + Math.random() * 84, y: 8 + Math.random() * 84 };
      });
      crystalsRef.current = nextCrystals;
      playersRef.current = { ...old, [playerId]: moved };
      setCrystals(nextCrystals); setPlayers(playersRef.current);
  }, [demo, matchEnded, playerId, send]);

  const nextLevel = () => {
    const next = level + 1;
    deadline.current = Date.now() + levelDuration(next);
    setLevel(next);
    setRemaining(levelDuration(next));
    setCrystals(crystalsForLevel(next));
    setPlayers(old => {
      const updated = { ...old };
      if (updated[playerId]) updated[playerId] = { ...updated[playerId], x: 42, y: 58 };
      if (updated.rival) updated.rival = { ...updated.rival, x: 70, y: 38, score: updated.rival.score + 10 };
      return updated;
    });
    setStatus(`Level ${next} started`);
    setMatchEnded(false);
  };

  useEffect(() => {
    if (!demo || screen !== "game" || matchEnded) return;
    const timer = window.setInterval(() => {
      const value = Math.max(0, deadline.current - Date.now());
      setRemaining(value);
      if (!value) { setMatchEnded(true); setStatus(`Level ${level} complete`); }
    }, 50);
    return () => window.clearInterval(timer);
  }, [demo, level, matchEnded, screen]);

  useEffect(() => {
    if (screen !== "game" || matchEnded) return;
    const keys = new Set<string>();
    const move = () => {
      const dx = touchDirection.current[0] || Number(keys.has("d") || keys.has("arrowright")) - Number(keys.has("a") || keys.has("arrowleft"));
      const dy = touchDirection.current[1] || Number(keys.has("s") || keys.has("arrowdown")) - Number(keys.has("w") || keys.has("arrowup"));
      movePlayer(dx, dy);
    };
    const interval = window.setInterval(move, 40);
    const movementKeys = new Set(["w", "a", "s", "d", "arrowup", "arrowdown", "arrowleft", "arrowright"]);
    const down = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if (!movementKeys.has(key)) return;
      event.preventDefault();
      keys.add(key);
    };
    const up = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if (!movementKeys.has(key)) return;
      event.preventDefault();
      keys.delete(key);
      move();
    };
    const blur = () => { keys.clear(); touchDirection.current = [0, 0]; move(); };
    window.addEventListener("blur", blur);
    window.addEventListener("keydown", down); window.addEventListener("keyup", up);
    return () => { window.clearInterval(interval); window.removeEventListener("keydown", down); window.removeEventListener("keyup", up); window.removeEventListener("blur", blur); };
  }, [matchEnded, movePlayer, screen]);

  if (screen === "start") return <StartScreen name={name} setName={setName} roomCode={roomCode} setRoomCode={setRoomCode} apiUrl={apiUrl} setApiUrl={setApiUrl} status={status} createRoom={createRoom} joinRoom={joinRoom} startDemo={startDemo} />;

  return (
    <main className="min-h-screen bg-[#08090d] p-3 text-white sm:p-6">
      <header className="mx-auto mb-4 flex max-w-[1500px] items-center justify-between rounded-2xl border border-white/10 bg-white/[0.035] px-4 py-3 backdrop-blur-xl">
        <div className="flex items-center gap-3"><div className="brand-mark"><Gem size={20}/></div><div><p className="font-display text-lg tracking-[0.16em]">OBSIDIAN ARENA</p><p className="text-xs text-zinc-500">ROOM {roomCode}</p></div></div>
        <div className="flex items-center gap-2"><Badge className="border-cyan-400/20 bg-cyan-400/10 text-cyan-300"><Radio size={12}/> {demo ? "SIMULATION" : "LIVE"}</Badge><Button variant="outline" size="sm" onClick={() => navigator.clipboard?.writeText(roomCode)}><Copy size={14}/> Copy code</Button></div>
      </header>
      {screen === "lobby" ? (
        <section className="mx-auto grid min-h-[72vh] max-w-5xl place-items-center"><div className="panel w-full max-w-xl p-8 text-center"><Users className="mx-auto mb-4 text-cyan-300" size={38}/><p className="eyebrow">WAITING CHAMBER</p><h1 className="font-display mt-2 text-4xl">Assemble your squad</h1><p className="mt-3 text-zinc-400">Share room code <strong className="text-white">{roomCode}</strong>. A match needs at least two ready players.</p><div className="mt-8 grid gap-3">{Object.values(players).map(player => <div key={player.id} className="flex items-center justify-between rounded-xl border border-white/8 bg-white/[0.025] p-4"><span>{player.name}</span><Badge variant="outline">{player.ready ? "Ready" : "Waiting"}</Badge></div>)}</div><div className="mt-8 flex justify-center gap-3"><Button onClick={() => send({ type: "ready", ready: true })}>Ready up</Button><Button variant="outline" onClick={() => send({ type: "start" })}>Start match</Button></div></div></section>
      ) : (
        <section className="mx-auto grid max-w-[1500px] gap-4 lg:grid-cols-[260px_minmax(0,1fr)_280px]">
          <aside className="panel order-2 p-5 lg:order-1"><p className="eyebrow">MATCH STATUS</p><div className="mt-5 space-y-5"><Stat icon={<Crown/>} label="Current level" value={String(level)}/><Stat icon={<Activity/>} label="Time remaining" value={`${Math.ceil(remaining/1000)}s`}/><Progress value={(remaining/levelDuration(level))*100}/><Stat icon={<Gem/>} label="Your score" value={String(currentPlayer?.score ?? 0)}/><Stat icon={<Swords/>} label="Players" value={`${Object.keys(players).length}/4`}/></div><div className="mt-8 rounded-xl border border-white/8 bg-black/20 p-4"><Keyboard size={20} className="mb-2 text-violet-300"/><p className="text-sm font-medium">Move with WASD or arrows</p><p className="mt-1 text-xs leading-5 text-zinc-400">Collect the bright cyan crystals. Each one gives you 10 points.</p><div className="dpad mt-4" aria-label="Movement controls"><button aria-label="Move up" disabled={matchEnded} onPointerDown={e=>{ e.currentTarget.setPointerCapture(e.pointerId); touchDirection.current = [0,-1]; }} onPointerUp={()=>{ touchDirection.current = [0,0]; movePlayer(0,0); }} onLostPointerCapture={()=>{ touchDirection.current = [0,0]; movePlayer(0,0); }}><ArrowUp/></button><button aria-label="Move left" disabled={matchEnded} onPointerDown={e=>{ e.currentTarget.setPointerCapture(e.pointerId); touchDirection.current = [-1,0]; }} onPointerUp={()=>{ touchDirection.current = [0,0]; movePlayer(0,0); }} onLostPointerCapture={()=>{ touchDirection.current = [0,0]; movePlayer(0,0); }}><ArrowLeft/></button><button aria-label="Move down" disabled={matchEnded} onPointerDown={e=>{ e.currentTarget.setPointerCapture(e.pointerId); touchDirection.current = [0,1]; }} onPointerUp={()=>{ touchDirection.current = [0,0]; movePlayer(0,0); }} onLostPointerCapture={()=>{ touchDirection.current = [0,0]; movePlayer(0,0); }}><ArrowDown/></button><button aria-label="Move right" disabled={matchEnded} onPointerDown={e=>{ e.currentTarget.setPointerCapture(e.pointerId); touchDirection.current = [1,0]; }} onPointerUp={()=>{ touchDirection.current = [0,0]; movePlayer(0,0); }} onLostPointerCapture={()=>{ touchDirection.current = [0,0]; movePlayer(0,0); }}><ArrowRight/></button></div></div></aside>
          <div className="arena order-1 aspect-[16/10] min-h-[430px] lg:order-2"><div className="arena-grid"/><div className="objective"><Gem size={16}/><strong>LEVEL {level} · COLLECT CRYSTALS</strong><span>+10 POINTS</span></div>{crystals.map(crystal => <div key={crystal.id} className="crystal" style={{ left:`${crystal.x}%`, top:`${crystal.y}%` }}><Gem size={24}/><span>+{crystal.value}</span></div>)}{Object.values(players).map(player => <div key={player.id} className={`player ${player.id === playerId ? "player-you" : "player-rival"}`} style={{ left:`${player.x}%`, top:`${player.y}%` }}><b>{player.id === playerId ? "YOU" : "RIVAL"}</b><span>{player.name}</span></div>)}{matchEnded && <div className="match-over"><div><p className="eyebrow">TIME IS UP</p><h2 className="font-display">{demo ? `Level ${level} Complete` : "Match finished"}</h2><p>You scored <strong>{currentPlayer?.score ?? 0}</strong> points.</p>{demo ? <Button onClick={nextLevel}>Play Level {level + 1}</Button> : <Button onClick={()=>{ const old = socketRef.current; socketRef.current = null; old?.close(); setScreen("start"); }}>Return to menu</Button>}</div></div>}<div className="arena-vignette"/></div>
          <aside className="panel order-3 p-5"><div className="flex items-center justify-between"><p className="eyebrow">LEADERBOARD</p><Crown size={18} className="text-amber-300"/></div><div className="mt-5 space-y-2">{Object.values(players).sort((a,b)=>b.score-a.score).map((player,index)=><div key={player.id} className="flex items-center gap-3 rounded-xl border border-white/8 bg-white/[0.025] p-3"><span className="rank">{index+1}</span><div className="min-w-0 flex-1"><p className="truncate text-sm font-medium">{player.name}</p><p className="text-xs text-zinc-500">Crystal hunter</p></div><strong className="text-cyan-300">{player.score}</strong></div>)}</div><p className="mt-6 text-xs leading-5 text-zinc-500">{status}</p></aside>
        </section>
      )}
    </main>
  );
}

function StartScreen(props: { name:string; setName:(v:string)=>void; roomCode:string; setRoomCode:(v:string)=>void; apiUrl:string; setApiUrl:(v:string)=>void; status:string; createRoom:()=>void; joinRoom:()=>void; startDemo:()=>void }) {
  return <main className="start-shell min-h-screen text-white"><div className="noise"/><section className="relative z-10 mx-auto grid min-h-screen max-w-7xl items-center gap-14 px-5 py-12 lg:grid-cols-[1.1fr_0.9fr] lg:px-10"><div><Badge className="mb-6 border-violet-400/25 bg-violet-400/10 text-violet-200"><Shield size={13}/> SERVER-AUTHORITATIVE</Badge><h1 className="font-display max-w-3xl text-6xl leading-[0.9] tracking-tight sm:text-8xl">ENTER THE<br/><span className="gradient-text">OBSIDIAN</span><br/>ARENA</h1><p className="mt-7 max-w-xl text-lg leading-8 text-zinc-400">Collect volatile crystals. Outsmart rival players. Every movement and point is verified by the Go game server.</p><div className="mt-8 flex flex-wrap gap-6 text-sm text-zinc-500"><span className="flex items-center gap-2"><Users size={16}/> 2–4 players</span><span className="flex items-center gap-2"><Activity size={16}/> 20 Hz simulation</span><span className="flex items-center gap-2"><Gem size={16}/> 10 points each</span></div></div><div className="panel glow-card p-6 sm:p-8"><p className="eyebrow">PLAYER TERMINAL</p><h2 className="font-display mt-2 text-3xl">Prepare for deployment</h2><div className="mt-7 space-y-4"><label className="field-label">Display name<Input value={props.name} onChange={e=>props.setName(e.target.value)} maxLength={24}/></label><label className="field-label">Backend URL<Input value={props.apiUrl} onChange={e=>props.setApiUrl(e.target.value)}/></label><Button className="h-12 w-full" onClick={props.createRoom}><Swords/> Create multiplayer room</Button><div className="divider"><span>OR JOIN</span></div><div className="flex gap-2"><Input placeholder="ROOM CODE" value={props.roomCode} onChange={e=>props.setRoomCode(e.target.value.toUpperCase())}/><Button variant="outline" onClick={props.joinRoom}>Join</Button></div><Button variant="ghost" className="w-full text-cyan-300" onClick={props.startDemo}>Play instant demo</Button></div><p className="mt-5 text-center text-xs text-zinc-500">{props.status}</p></div></section></main>;
}

function Stat({icon,label,value}:{icon:React.ReactNode;label:string;value:string}) { return <div className="flex items-center gap-3"><span className="stat-icon">{icon}</span><div><p className="text-xs text-zinc-500">{label}</p><p className="font-display text-2xl">{value}</p></div></div>; }
