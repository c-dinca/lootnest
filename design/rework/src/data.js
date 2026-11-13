const ART = /*@ART@*/{};
const A = (n) => ART[n] || "";
const GEM = `<svg class="gem" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M2.75 10.25c0 3.7 3.25 6.25 7.25 6.25s7.25-2.55 7.25-6.25"/><path d="M5.75 10.25c0 2.2 1.9 3.5 4.25 3.5s4.25-1.3 4.25-3.5"/><circle cx="10" cy="6.25" r="2.5" stroke="#ef233c"/></svg>`;
const PLAY = `<svg viewBox="0 0 20 20"><path d="M5 3.5v13l11-6.5z" fill="currentColor"/></svg>`;
const BACK = `<svg viewBox="0 0 20 20" width="16" height="16"><path d="M12.5 4 6.5 10l6 6" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>`;
const SEARCH = `<svg viewBox="0 0 20 20" width="16" height="16"><circle cx="9" cy="9" r="5.5" fill="none" stroke="currentColor" stroke-width="1.7"/><path d="m13.2 13.2 3.3 3.3" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/></svg>`;
const MORE = `<svg viewBox="0 0 20 20" width="16" height="16"><circle cx="4.5" cy="10" r="1.6" fill="currentColor"/><circle cx="10" cy="10" r="1.6" fill="currentColor"/><circle cx="15.5" cy="10" r="1.6" fill="currentColor"/></svg>`;
const PROFILES = [
  { id: "p1", name: "Player 1", short: "P1", c: "#6c9bd2" },
  { id: "p2", name: "Player 2", short: "P2", c: "#e88ca0" },
  { id: "run", name: "Player 1, hardcore", short: "H", c: "#6c9bd2", of: "p1" },
];
const GAMES = [
  { id: "bg3", name: "Baldur's Gate 3", store: "Steam", h: 132, last: "Played 2 days ago", ach: [11, 54], status: "Playing", line: "Act 3, Lower City", dev: "Larian Studios", year: 2023, saves: 6, points: 3 },
  { id: "cs2", name: "Counter-Strike 2", store: "Steam", h: 412, last: "Played 5 hours ago", ach: [1, 1], status: "Playing", line: "Premier, rank 14", dev: "Valve", year: 2023, saves: 2, points: 0 },
  { id: "ittakestwo", name: "It Takes Two", store: "EA app", h: 99, last: "Played yesterday", ach: [14, 20], status: "Up next", line: "Chapter 4, Rose's Room", dev: "Hazelight", year: 2021, saves: 4, points: 1 },
  { id: "hollow", name: "Hollow Knight", store: "In a folder", h: 41, last: "Played last week", ach: [37, 63], status: "Playing", line: "City of Tears", dev: "Team Cherry", year: 2017, saves: 9, points: 5 },
  { id: "cyberpunk", name: "Cyberpunk 2077", store: "GOG", h: 84, last: "Played 3 weeks ago", ach: [22, 57], status: "Dropped", line: "Act 2", dev: "CD Projekt Red", year: 2020, saves: 12, points: 2 },
  { id: "hades2", name: "Hades II", store: "Epic", h: 27, last: "Played a month ago", ach: [9, 49], status: "Up next", line: "Night 12", dev: "Supergiant", year: 2025, saves: 3, points: 0 },
  { id: "stardew", name: "Stardew Valley", store: "GOG", h: 210, last: "Played 2 months ago", ach: [30, 49], status: "Finished", line: "Year 3, Spring", dev: "ConcernedApe", year: 2016, saves: 7, points: 1 },
  { id: "forza", name: "Forza Horizon 5", store: "Xbox", h: 33, last: "Played in March", ach: [40, 175], status: "", line: "Festival", dev: "Playground", year: 2021, saves: 1, points: 0 },
  { id: "gtav", name: "Grand Theft Auto V", store: "Rockstar", h: 160, last: "Played last year", ach: [60, 77], status: "Finished", line: "Story done", dev: "Rockstar North", year: 2013, saves: 5, points: 0 },
  { id: "isaac", name: "The Binding of Isaac", store: "Steam", h: 133, last: "Played last year", ach: [400, 637], status: "", line: "Run 812", dev: "Edmund McMillen", year: 2014, saves: 1, points: 0 },
  { id: "forest", name: "The Forest", store: "Steam", h: 18, last: "Played in 2025", ach: [12, 46], status: "", line: "Day 9", dev: "Endnight", year: 2018, saves: 2, points: 0 },
];
const MODS = [["Better camera", "12 files", true], ["Bigger party", "3 files", true], ["Fast travel anywhere", "1 file", false]];
const POINTS = [["Before the boss", "Today, 21:14", "Act 3, Lower City"], ["Sold the sword", "Yesterday", "Act 2, Moonrise Towers"], ["Start of Act 2", "Last week", "Act 2, Shadow-Cursed Lands"]];
const ACH = [["Critical hit", "Rare, 4.1% of players", true], ["Descent from Avernus", "62% of players", true], ["Hope endures", "Ultra rare, 0.8%", false], ["Pest control", "31% of players", true]];
