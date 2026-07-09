// ScoresComponent: React component that subscribes to the scores-service SSE
// stream and renders a live leaderboard standings list.
// Uses React.createElement only (no JSX, no build step required).
// React and ReactDOM are loaded as UMD bundles before this module is imported.

const { useState, useEffect } = React;

const POLL_INTERVAL_MS = 5000;

// ScoresComponent renders the standings column contents.
// Props: { scoresServiceURL: string }
function ScoresComponent({ scoresServiceURL }) {
  // null = loading (no data yet), [] = loaded but empty, [...] = has standings
  const [standings, setStandings] = useState(null);

  useEffect(() => {
    let cleanup = null;

    if (typeof EventSource !== 'undefined') {
      cleanup = startSSE(scoresServiceURL, setStandings);
    } else {
      cleanup = startPolling(scoresServiceURL, setStandings);
    }

    return cleanup;
  }, [scoresServiceURL]);

  if (standings === null) {
    return null;
  }

  if (standings.length === 0) {
    return React.createElement(
      'p',
      null,
      'No scores yet — be the first to play!'
    );
  }

  return React.createElement(
    'ul',
    null,
    standings.map(function (s) {
      return React.createElement(
        'li',
        { key: s.rank + '-' + s.name },
        React.createElement('span', { className: 'rank' }, '#' + s.rank),
        React.createElement('span', { className: 'name' }, s.name),
        React.createElement('span', { className: 'score' }, s.score)
      );
    })
  );
}

// startSSE opens an EventSource connection to the scores stream.
// On each "standings" event, updates state. On permanent error, falls back to
// polling. Returns a cleanup function.
function startSSE(baseURL, setStandings) {
  const source = new EventSource(baseURL + '/scores/stream');
  let pollingCleanup = null;

  source.addEventListener('standings', function (e) {
    try {
      const data = JSON.parse(e.data);
      setStandings(data.standings || []);
    } catch (_) {}
  });

  source.onerror = function () {
    if (source.readyState === EventSource.CLOSED) {
      source.close();
      pollingCleanup = startPolling(baseURL, setStandings);
    }
  };

  return function () {
    source.close();
    if (pollingCleanup) pollingCleanup();
  };
}

// startPolling fetches the standings on a fixed interval.
// Returns a cleanup function.
function startPolling(baseURL, setStandings) {
  function fetchStandings() {
    fetch(baseURL + '/scores')
      .then(function (resp) {
        if (!resp.ok) throw new Error('scores fetch failed: ' + resp.status);
        return resp.json();
      })
      .then(function (data) {
        setStandings(data.standings || []);
      })
      .catch(function () {
        // Leave last known state intact on transient failure.
      });
  }

  fetchStandings();
  const id = setInterval(fetchStandings, POLL_INTERVAL_MS);
  return function () { clearInterval(id); };
}

export { ScoresComponent };
