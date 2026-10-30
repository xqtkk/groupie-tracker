(() => {
  const input = document.querySelector('#artist-search');
  const grid = document.querySelector('#artist-grid');
  if (!input || !grid) return;

  const cards = [...grid.querySelectorAll('.artist-card')];
  const suggestions = document.querySelector('#search-suggestions');
  const resultCount = document.querySelector('#result-count');
  const emptyState = document.querySelector('#empty-state');
  const locationSearch = document.querySelector('#location-search');
  const locationOptions = document.querySelector('#location-options');

  const filters = {
    creationMin: document.querySelector('#creation-min'),
    creationMax: document.querySelector('#creation-max'),
    albumMin: document.querySelector('#album-min'),
    albumMax: document.querySelector('#album-max'),
    membersMin: document.querySelector('#members-min'),
    membersMax: document.querySelector('#members-max'),
  };

  const artists = cards.map((card) => ({
    card,
    name: card.dataset.name || '',
    members: JSON.parse(card.dataset.members || '[]') || [],
    creation: Number(card.dataset.creation || 0),
    album: card.dataset.album || '',
    albumYear: Number(card.dataset.albumYear || 0),
    memberCount: Number(card.dataset.memberCount || 0),
    locations: JSON.parse(card.dataset.locations || '[]') || [],
  }));

  const searchIndex = [];
  const seenSuggestions = new Set();
  const addSuggestion = (value, type, artist) => {
    const clean = String(value || '').trim();
    if (!clean) return;
    const key = `${type}:${clean.toLocaleLowerCase()}:${artist}`;
    if (seenSuggestions.has(key)) return;
    seenSuggestions.add(key);
    searchIndex.push({ value: clean, type, artist });
  };

  for (const artist of artists) {
    addSuggestion(artist.name, 'Artist / band', artist.name);
    for (const member of artist.members) addSuggestion(member, 'Member', artist.name);
    for (const location of artist.locations) addSuggestion(location, 'Concert location', artist.name);
    addSuggestion(artist.album, 'First album date', artist.name);
    addSuggestion(String(artist.creation), 'Formation year', artist.name);
  }

  const normalize = (value) => String(value || '').toLocaleLowerCase().replaceAll('_', ' ').replaceAll('-', ' ').trim();
  const numberFrom = (element) => element && element.value !== '' ? Number(element.value) : null;
  const inside = (value, low, high) => (low === null || value >= low) && (high === null || value <= high);

  function matchesLocation(location, selected) {
    const candidate = normalize(location);
    return selected.some((value) => {
      const target = normalize(value);
      return candidate === target || candidate.endsWith(` ${target}`);
    });
  }

  function applyFilters() {
    const query = normalize(input.value);
    const creationMin = numberFrom(filters.creationMin);
    const creationMax = numberFrom(filters.creationMax);
    const albumMin = numberFrom(filters.albumMin);
    const albumMax = numberFrom(filters.albumMax);
    const membersMin = numberFrom(filters.membersMin);
    const membersMax = numberFrom(filters.membersMax);
    const selectedLocations = [...document.querySelectorAll('#location-options input:checked')].map((box) => box.value);
    let visible = 0;

    for (const artist of artists) {
      const searchable = normalize([
        artist.name,
        ...artist.members,
        artist.creation,
        artist.album,
        ...artist.locations,
      ].join(' '));
      const searchMatch = !query || searchable.includes(query);
      const creationMatch = inside(artist.creation, creationMin, creationMax);
      const albumMatch = (albumMin === null && albumMax === null) || (artist.albumYear > 0 && inside(artist.albumYear, albumMin, albumMax));
      const membersMatch = inside(artist.memberCount, membersMin, membersMax);
      const locationMatch = selectedLocations.length === 0 || artist.locations.some((location) => matchesLocation(location, selectedLocations));
      const show = searchMatch && creationMatch && albumMatch && membersMatch && locationMatch;
      artist.card.hidden = !show;
      if (show) visible++;
    }

    resultCount.textContent = `${visible} ${visible === 1 ? 'artist' : 'artists'}`;
    emptyState.hidden = visible !== 0;
  }

  function closeSuggestions() {
    suggestions.hidden = true;
    input.setAttribute('aria-expanded', 'false');
  }

  function showSuggestions() {
    const query = normalize(input.value);
    suggestions.replaceChildren();
    if (!query) {
      closeSuggestions();
      return;
    }

    const matches = searchIndex.filter((item) => normalize(item.value).includes(query)).slice(0, 8);
    if (matches.length === 0) {
      closeSuggestions();
      return;
    }

    for (const item of matches) {
      const option = document.createElement('button');
      option.type = 'button';
      option.className = 'suggestion-option';
      option.setAttribute('role', 'option');
      const value = document.createElement('span');
      value.className = 'suggestion-value';
      value.textContent = item.value;
      const meta = document.createElement('span');
      meta.className = 'suggestion-meta';
      meta.textContent = `${item.type} · ${item.artist}`;
      option.append(value, meta);
      option.addEventListener('mousedown', (event) => event.preventDefault());
      option.addEventListener('click', () => {
        input.value = item.value;
        closeSuggestions();
        applyFilters();
      });
      suggestions.append(option);
    }
    suggestions.hidden = false;
    input.setAttribute('aria-expanded', 'true');
  }

  input.addEventListener('input', () => {
    showSuggestions();
    applyFilters();
  });
  input.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') closeSuggestions();
    if (event.key === 'Enter' && !suggestions.hidden) {
      const first = suggestions.querySelector('button');
      if (first) {
        event.preventDefault();
        first.click();
      }
    }
  });
  input.addEventListener('blur', () => window.setTimeout(closeSuggestions, 120));

  for (const control of Object.values(filters)) control.addEventListener('input', applyFilters);
  for (const checkbox of document.querySelectorAll('#location-options input[type="checkbox"]')) checkbox.addEventListener('change', applyFilters);

  locationSearch.addEventListener('input', () => {
    const query = normalize(locationSearch.value);
    for (const option of locationOptions.querySelectorAll('.location-option')) {
      option.hidden = !normalize(option.textContent).includes(query);
    }
  });

  document.querySelector('#clear-filters').addEventListener('click', () => {
    input.value = '';
    for (const control of Object.values(filters)) control.value = '';
    for (const checkbox of document.querySelectorAll('#location-options input[type="checkbox"]')) checkbox.checked = false;
    locationSearch.value = '';
    for (const option of locationOptions.querySelectorAll('.location-option')) option.hidden = false;
    closeSuggestions();
    applyFilters();
  });
})();
