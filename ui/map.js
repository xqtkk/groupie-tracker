(() => {
  const button = document.querySelector('#load-map');
  const container = document.querySelector('#concert-map');
  const status = document.querySelector('#map-status');
  if (!button || !container || !status) return;

  let map;

  const locationName = (value) => value.replaceAll('_', ' ').replaceAll('-', ', ');

  function popupFor(point) {
    const content = document.createElement('div');
    const title = document.createElement('strong');
    title.textContent = locationName(point.location);
    content.append(title);
    if (point.dates && point.dates.length) {
      const dates = document.createElement('ul');
      for (const date of point.dates) {
        const item = document.createElement('li');
        item.textContent = date.replace(/^\*+/, '');
        dates.append(item);
      }
      content.append(dates);
    }
    return content;
  }

  function renderMap(points) {
    if (!window.L) throw new Error('Map library did not load. Check your internet connection and reload this page.');
    if (map) map.remove();
    map = L.map(container, { scrollWheelZoom: false });
    L.tileLayer(container.dataset.mapTiles || 'https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19,
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap contributors</a>',
    }).addTo(map);

    const bounds = [];
    for (const point of points) {
      const lat = Number(point.latitude);
      const lon = Number(point.longitude);
      if (!Number.isFinite(lat) || !Number.isFinite(lon)) continue;
      const marker = L.marker([lat, lon]).addTo(map);
      marker.bindPopup(popupFor(point));
      bounds.push([lat, lon]);
    }
    if (bounds.length === 1) map.setView(bounds[0], 5);
    else if (bounds.length > 1) map.fitBounds(bounds, { padding: [30, 30], maxZoom: 5 });
    window.setTimeout(() => map.invalidateSize(), 0);
  }

  button.addEventListener('click', async () => {
    button.disabled = true;
    container.hidden = false;
    status.textContent = 'Finding concert coordinates. The first map can take a little while.';
    try {
      const response = await fetch(`/api/artist-locations?id=${encodeURIComponent(button.dataset.artistId)}`, {
        headers: { Accept: 'application/json' },
      });
      if (!response.ok) throw new Error(await response.text() || 'Could not load map locations.');
      const data = await response.json();
      if (!data.points || data.points.length === 0) {
        container.hidden = true;
        status.textContent = 'No concert locations could be placed on the map.';
        return;
      }
      renderMap(data.points);
      const missing = data.unlocated ? data.unlocated.length : 0;
      status.textContent = missing
        ? `${data.points.length} locations shown. ${missing} could not be matched to coordinates.`
        : `${data.points.length} concert locations shown.`;
    } catch (error) {
      container.hidden = true;
      status.textContent = error.message || 'The map is temporarily unavailable.';
    } finally {
      button.disabled = false;
    }
  });
})();
