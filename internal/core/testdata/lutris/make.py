# Writes pga.db, a Lutris library for core's tests, with Python's own
# SQLite: run `python3 make.py` in this folder. The games table is created
# as Lutris's schema.py does; small pages, many rows and one long row make
# the reader walk interior pages and overflow pages, and discord_id is added
# afterwards, as Lutris's migrate() does, so older rows lack it.
import os
import sqlite3

if os.path.exists("pga.db"):
    os.remove("pga.db")
db = sqlite3.connect("pga.db")
db.execute("PRAGMA page_size = 1024")
db.execute(
    "CREATE TABLE IF NOT EXISTS games (id INTEGER PRIMARY KEY, name TEXT, sortname TEXT, slug TEXT, "
    "installer_slug TEXT, parent_slug TEXT, platform TEXT, runner TEXT, executable TEXT, directory TEXT, "
    "updated DATETIME, lastplayed INTEGER, installed INTEGER, installed_at INTEGER, year INTEGER, "
    "configpath TEXT, has_custom_banner INTEGER, has_custom_icon INTEGER, has_custom_coverart_big INTEGER, "
    "playtime REAL, service TEXT, service_id TEXT)"
)
ins = "INSERT INTO games (name, slug, runner, platform, installed, year, configpath, playtime, lastplayed) VALUES (?,?,?,?,?,?,?,?,?)"
for i in range(400):
    db.execute(ins, ("Filler %03d" % i, "filler-%03d" % i, "wine", "Windows", 0, 2000 + i % 20, None, 0.0, 0))
db.execute(ins, ("Ashen Ring", "ashen-ring", "wine", "Windows", 1, 2024, "ashen-ring-1700000000", 12.5, 1700000123456))
db.execute(ins, ("Tux Racer", "tux-racer", "linux", "Linux", 1, -12, "tux-racer-1700000001", 1.25, 0))
db.execute(ins, ("Steam Game", "steam-game", "steam", "Linux", 1, 2020, "steam-game-1", 0.0, 0))
db.execute(ins, ("Removed Game", "removed-game", "wine", "Windows", 0, 2021, "removed-game-1", 0.0, 0))
db.execute(ins, ("Long " + "x" * 3000, "long", "linux", "Linux", 0, 1999, None, 0.0, 0))
db.execute("ALTER TABLE games ADD COLUMN discord_id TEXT")
db.execute(ins.replace("lastplayed)", "lastplayed, discord_id)").replace("?)", "?,?)"),
           ("Newer Game", "newer-game", "wine", "Windows", 0, 2026, None, 0.0, 0, "123"))
db.commit()
db.execute("VACUUM")
db.close()
