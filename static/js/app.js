// BilimQuiz — global JS utilities

// Alpine component: lang modal
function langModal() {
  return {
    init() {
      if (!document.cookie.includes('lang=')) {
        const code = (navigator.language || '').split('-')[0].toLowerCase();
        if (!['ky', 'ru', 'en'].includes(code)) {
          this.$el.classList.remove('hidden');
        }
      }
    }
  };
}

// Avatar upload via Canvas API (compress to 120x120, ~2KB)
function compressAndUpload(file, callback) {
  const reader = new FileReader();
  reader.onload = function(e) {
    const img = new Image();
    img.onload = function() {
      const canvas = document.createElement('canvas');
      canvas.width = 120;
      canvas.height = 120;
      const ctx = canvas.getContext('2d');

      // crop square from center
      const size = Math.min(img.width, img.height);
      const sx = (img.width - size) / 2;
      const sy = (img.height - size) / 2;
      ctx.drawImage(img, sx, sy, size, size, 0, 0, 120, 120);

      // compress: try quality from 0.85 down until ~2KB
      let quality = 0.85;
      let dataURI = canvas.toDataURL('image/jpeg', quality);
      while (dataURI.length > 2800 && quality > 0.1) {
        quality -= 0.1;
        dataURI = canvas.toDataURL('image/jpeg', quality);
      }

      fetch('/upload/avatar', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: 'image_data=' + encodeURIComponent(dataURI),
      })
        .then(r => r.json())
        .then(data => callback(data.url))
        .catch(() => callback(null));
    };
    img.src = e.target.result;
  };
  reader.readAsDataURL(file);
}

// WebSocket helper used in lobby/gameplay pages
function createWS(path, onMessage) {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const ws = new WebSocket(`${proto}://${location.host}${path}`);
  ws.onmessage = e => {
    try { onMessage(JSON.parse(e.data)); } catch {}
  };
  ws.onerror = e => console.error('WS error', e);
  return ws;
}
