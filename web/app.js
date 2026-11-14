'use strict';

const SPEAKER_SVG =
  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon><path d="M15.5 8.5a5 5 0 0 1 0 7"></path><path d="M18.5 5.5a9 9 0 0 1 0 13"></path></svg>';

const els = {
  livePill: document.getElementById('livePill'),
  demoPill: document.getElementById('demoPill'),
  langSeg: document.getElementById('langSeg'),
  modelSelect: document.getElementById('modelSelect'),
  orb: document.getElementById('orb'),
  orbHint: document.getElementById('orbHint'),
  status: document.getElementById('status'),
  topicLine: document.getElementById('topicLine'),
  topicText: document.getElementById('topicText'),
  transcript: document.getElementById('transcript'),
  startBtn: document.getElementById('startBtn'),
  endBtn: document.getElementById('endBtn'),
  micDock: document.getElementById('micDock'),
  micBtn: document.getElementById('micBtn'),
  micLabel: document.getElementById('micLabel'),
  typeForm: document.getElementById('typeForm'),
  typeInput: document.getElementById('typeInput'),
};

const state = {
  lang: 'en',
  model: 'deepseek-v4-pro',
  sessionId: null,
  active: false,
  listening: false,
  speaking: false,
  thinking: false,
  demo: false,
  models: [],
  audio: null,
  recognition: null,
  srSupported: false,
};

const STT_FALLBACK = { en: 'en-US', es: 'es-ES', fr: 'fr-FR' };

async function api(path, body) {
  const res = await fetch(path, {
    method: body ? 'POST' : 'GET',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`);
  return data;
}

/* ---- Setup / rendering -------------------------------------------- */

async function loadLanguages() {
  const { languages } = await api('/api/languages');
  els.langSeg.innerHTML = '';
  languages.forEach((l) => {
    const b = document.createElement('button');
    b.type = 'button';
    b.className = 'segment';
    b.dataset.code = l.code;
    b.textContent = l.native;
    b.setAttribute('role', 'tab');
    b.addEventListener('click', () => setLang(l.code));
    els.langSeg.appendChild(b);
  });
  setLang(languages[0]?.code || 'en');
}

async function loadModels() {
  const { models, default: def } = await api('/api/models');
  state.models = models;
  els.modelSelect.innerHTML = '';
  models.forEach((m) => {
    const o = document.createElement('option');
    o.value = m.id;
    o.textContent = m.available ? m.label : `${m.label} (no key · demo)`;
    els.modelSelect.appendChild(o);
  });
  const pick = models.some((m) => m.id === def) ? def : (models[0]?.id || '');
  els.modelSelect.value = pick;
  state.model = pick;
  els.modelSelect.addEventListener('change', () => {
    state.model = els.modelSelect.value;
  });
}

// A model is in demo mode when its provider has no credentials configured
// (the server then falls back to the offline demo tutor).
function modelDemoMode(id) {
  const m = (state.models || []).find((x) => x.id === id);
  return !m || !m.available;
}

function setLang(code) {
  state.lang = code;
  const lang = (els.langSeg.querySelector(`[data-code="${code}"]`));
  els.orbHint.textContent = lang ? lang.textContent : 'English';
  [...els.langSeg.children].forEach((b) => {
    b.classList.toggle('active', b.dataset.code === code);
    b.setAttribute('aria-selected', b.dataset.code === code ? 'true' : 'false');
  });
}

function checkSpeechSupport() {
  state.srSupported = !!(window.SpeechRecognition || window.webkitSpeechRecognition);
  els.micBtn.hidden = !state.srSupported;
}

/* ---- Orb / status / controls --------------------------------------- */

function syncOrb() {
  let s = 'idle';
  if (state.listening) s = 'listening';
  else if (state.speaking) s = 'speaking';
  else if (state.thinking) s = 'thinking';
  els.orb.dataset.state = s;
}

function setStatus(text) { els.status.textContent = text; }

function setListening(on) {
  state.listening = on;
  els.micBtn.classList.toggle('listening', on);
  els.micLabel.textContent = on ? 'Listening…' : 'Tap to speak';
  if (on) setStatus('Listening — speak now.');
  syncOrb();
}

function setSpeaking(on) {
  state.speaking = on;
  if (on) setStatus('Tutor is speaking…');
  syncOrb();
}

function setThinking(on) {
  state.thinking = on;
  syncOrb();
}

function setControlsActive() {
  els.startBtn.disabled = state.active;
  els.endBtn.disabled = !state.active;
  els.micDock.hidden = !state.active;
  [...els.langSeg.children].forEach((b) => (b.disabled = state.active));
  els.modelSelect.disabled = state.active;
}

function setLive(on) { els.livePill.hidden = !on; }
function updateDemoBadge() { els.demoPill.hidden = !state.demo; }
function showTopic(topic) { els.topicText.textContent = topic; els.topicLine.hidden = false; }
function hideTopic() { els.topicLine.hidden = true; }

/* ---- Transcript ----------------------------------------------------- */

function appendMessage(role, text, speakable) {
  if (els.transcript.querySelector('.empty')) els.transcript.innerHTML = '';

  const wrap = document.createElement('div');
  wrap.className = `msg msg-${role}`;

  const bubble = document.createElement('div');
  bubble.className = 'bubble';
  bubble.textContent = text;
  wrap.appendChild(bubble);

  if (role === 'assistant') {
    const actions = document.createElement('div');
    actions.className = 'msg-actions';
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'replay';
    btn.title = 'Replay';
    btn.innerHTML = SPEAKER_SVG;
    btn.addEventListener('click', () => { stopAudio(); speak(text); });
    actions.appendChild(btn);
    wrap.appendChild(actions);
  }

  els.transcript.appendChild(wrap);
  els.transcript.scrollTop = els.transcript.scrollHeight;
}

function renderEmptyState() {
  const div = document.createElement('div');
  div.className = 'empty';
  div.textContent = 'Your conversation will appear here.';
  els.transcript.appendChild(div);
}

/* ---- Session lifecycle --------------------------------------------- */

async function startDiscussion() {
  if (state.active) return;
  els.startBtn.disabled = true;
  setThinking(true);
  setStatus('Starting a new discussion…');
  try {
    const s = await api('/api/session/start', { language: state.lang, model: state.model });
    state.sessionId = s.sessionId;
    state.active = true;
    state.demo = modelDemoMode(s.model);
    updateDemoBadge();
    els.transcript.innerHTML = '';
    showTopic(s.topic);
    setLive(true);
    setControlsActive();
    appendMessage('assistant', s.greeting);
    await speak(s.greeting);
    setStatus('Tap the mic and speak, or type below.');
  } catch (err) {
    setStatus('Could not start: ' + err.message);
  } finally {
    setThinking(false);
    setControlsActive();
  }
}

async function endDiscussion() {
  if (!state.active || !state.sessionId) return;
  stopAudio();
  stopListening();
  setStatus('Ending discussion…');
  const sid = state.sessionId;
  state.active = false;
  state.sessionId = null;
  setLive(false);
  setControlsActive();
  hideTopic();
  try {
    const r = await api('/api/session/end', { sessionId: sid });
    appendMessage('assistant', r.goodbye);
  } catch {
    /* session may already be gone — no-op */
  }
  setStatus('Discussion ended. Start a new one whenever you like.');
}

/* ---- Voice input (speech recognition) ------------------------------ */

function toggleMic() {
  if (state.listening) { stopListening(); return; }
  if (state.speaking) stopAudio();
  startListening();
}

function startListening() {
  if (!state.active || !state.srSupported) return;
  const SR = window.SpeechRecognition || window.webkitSpeechRecognition;
  const rec = new SR();
  rec.continuous = false;
  rec.interimResults = true;
  rec.lang = STT_FALLBACK[state.lang] || 'en-US';

  rec.onresult = (e) => {
    let interim = '';
    let final = '';
    for (let i = e.resultIndex; i < e.results.length; i++) {
      const t = e.results[i][0].transcript;
      if (e.results[i].isFinal) final += t;
      else interim += t;
    }
    if (final) {
      stopListening();
      sendUserMessage(final.trim());
    } else if (interim) {
      setStatus('Heard: “' + interim + '”');
    }
  };

  rec.onend = () => {
    if (state.listening) {
      setListening(false);
      if (!state.thinking && !state.speaking) setStatus('Tap the mic and speak.');
    }
  };

  rec.onerror = (e) => {
    setListening(false);
    handleRecError(e.error);
  };

  state.recognition = rec;
  setListening(true);
  try {
    rec.start();
  } catch (err) {
    setListening(false);
    setStatus('Could not open the microphone: ' + err.message);
  }
}

function stopListening() {
  if (state.recognition) {
    try { state.recognition.stop(); } catch { /* no-op */ }
    state.recognition = null;
  }
  setListening(false);
}

function handleRecError(code) {
  const msgs = {
    'not-allowed': 'Microphone access was blocked. Allow the mic in your browser and try again.',
    'service-not-allowed': 'Microphone access was blocked. Allow the mic and try again.',
    'no-speech': 'Didn’t catch anything — tap the mic and try again.',
    'audio-capture': 'No microphone found. Check your mic and try again.',
    'network': 'Speech recognition needs an internet connection.',
    'language-not-supported': 'Your browser can’t recognize this language yet.',
  };
  setStatus(msgs[code] || 'Speech recognition error: ' + code);
}

/* ---- Send a turn ---------------------------------------------------- */

async function sendUserMessage(text) {
  if (!state.active || !state.sessionId) return;
  appendMessage('user', text);
  setThinking(true);
  setStatus('Tutor is thinking…');
  try {
    const r = await api('/api/session/message', { sessionId: state.sessionId, text });
    if (r.demo) { state.demo = true; updateDemoBadge(); }
    appendMessage('assistant', r.reply);
    await speak(r.reply);
    setStatus('Tap the mic and speak, or type below.');
  } catch (err) {
    appendMessage('system', 'Something went wrong: ' + err.message);
    setStatus('Something went wrong. Try again.');
  } finally {
    setThinking(false);
  }
}

/* ---- Server-side voice (TTS) --------------------------------------- */

async function speak(text) {
  stopAudio();
  setSpeaking(true);
  try {
    const res = await fetch('/api/tts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text, lang: state.lang }),
    });
    if (!res.ok) throw new Error('tts ' + res.status);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    await playAudio(url);
  } catch {
    browserSpeak(text);
  }
}

function playAudio(url) {
  return new Promise((resolve) => {
    const a = new Audio(url);
    state.audio = a;
    a.onended = () => { state.audio = null; setSpeaking(false); resolve(); };
    a.onerror = () => { state.audio = null; setSpeaking(false); resolve(); };
    a.play().catch(() => { state.audio = null; setSpeaking(false); resolve(); });
  });
}

function browserSpeak(text) {
  if (!('speechSynthesis' in window)) { setSpeaking(false); return; }
  const u = new SpeechSynthesisUtterance(text);
  u.lang = STT_FALLBACK[state.lang] || 'en-US';
  u.onend = () => setSpeaking(false);
  u.onerror = () => setSpeaking(false);
  try {
    speechSynthesis.cancel();
    speechSynthesis.speak(u);
  } catch {
    setSpeaking(false);
  }
}

function stopAudio() {
  if (state.audio) {
    try { state.audio.pause(); } catch { /* no-op */ }
    state.audio = null;
  }
  if ('speechSynthesis' in window) {
    try { speechSynthesis.cancel(); } catch { /* no-op */ }
  }
  setSpeaking(false);
}

/* ---- Wire-up -------------------------------------------------------- */

function bind() {
  els.startBtn.addEventListener('click', startDiscussion);
  els.endBtn.addEventListener('click', endDiscussion);
  els.micBtn.addEventListener('click', toggleMic);
  els.typeForm.addEventListener('submit', (e) => {
    e.preventDefault();
    const t = els.typeInput.value.trim();
    if (!t) return;
    els.typeInput.value = '';
    sendUserMessage(t);
  });
}

async function init() {
  bind();
  renderEmptyState();
  checkSpeechSupport();
  try {
    await Promise.all([loadLanguages(), loadModels()]);
  } catch (err) {
    setStatus('Could not reach the server: ' + err.message);
  }
}

init();
