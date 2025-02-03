import requests

owner = "pherrymason"
repo = "c3-lsp"
h = {"Accept": "application/vnd.github.v3+json"}
u = f"https://api.github.com/repos/{owner}/{repo}/releases?per_page=10"
r = requests.get(u, headers=h).json()
#r.reverse() # older tags first
for rel in r:
  if rel['assets']:
    tag = rel['tag_name']
    dls = rel['assets'][0]['download_count']
    pub = rel['published_at']
    print(f"Pub: {pub} | Tag: {tag} | Dls: {dls} ")