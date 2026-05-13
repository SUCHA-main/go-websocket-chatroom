const currentUserEl = document.getElementById('currentUser');
const logoutForm = document.getElementById('logoutForm');
const statusEl = document.getElementById('status');
const onlineCountEl = document.getElementById('onlineCount');
const messagesEl = document.getElementById('messages');
const messageInput = document.getElementById('messageInput');
const emojiBtn = document.getElementById('emojiBtn');
const sendBtn = document.getElementById('sendBtn');
const emojiPanel = document.getElementById('emojiPanel');
const stickerList = document.getElementById('stickerList');
const charCountEl = document.getElementById('charCount');

const maxMessageLength = 800;
const stickers = [1, 2, 3, 4, 5, 6].map((number) => `/stickers/sticker-${number}.svg`);
let socket = null;
let currentUser = '';

function setConnected(connected) {
    statusEl.textContent = connected ? '已连接' : '未连接';
    statusEl.classList.toggle('connected', connected);
    messageInput.disabled = !connected;
    emojiBtn.disabled = !connected;
    sendBtn.disabled = !connected;
}

function setOnlineCount(count) {
    onlineCountEl.textContent = `在线人数：${count || 0}`;
}

function updateCharCount() {
    const length = messageInput.value.length;
    charCountEl.textContent = `${length}/${maxMessageLength}`;
    charCountEl.classList.toggle('near-limit', length > maxMessageLength * 0.9);
}

function addMessage(data) {
    if (data.type === 'history') {
        addMessage({ type: 'history', username: '系统', content: data.content, time: data.time, onlineCount: data.onlineCount });
        (data.history || []).forEach(addMessage);
        return;
    }

    const item = document.createElement('div');
    item.className = 'message';
    if (data.type === 'system') item.classList.add('system');
    if (data.type === 'online') item.classList.add('online-message');
    if (data.type === 'history') item.classList.add('history-message');
    if (data.type === 'ai') item.classList.add('ai-message');
    if (data.username === currentUser) item.classList.add('own-message');
    if (data.username && data.username !== currentUser && !['系统', 'AI助手'].includes(data.username)) {
        item.classList.add('other-message');
    }

    const meta = document.createElement('div');
    meta.className = 'message-meta';
    const label = data.type === 'ai' ? 'AI助手' : (data.username || '匿名');
    meta.textContent = `${data.time || '--:--:--'} · ${label}`;
    item.appendChild(meta);

    if (data.type === 'sticker' && data.stickerUrl) {
        const img = document.createElement('img');
        img.className = 'sticker-image';
        img.src = data.stickerUrl;
        img.alt = data.content || '表情包';
        item.appendChild(img);
    } else {
        const content = document.createElement('div');
        content.className = 'message-content';
        content.textContent = data.content || '';
        item.appendChild(content);
    }

    messagesEl.appendChild(item);
    messagesEl.scrollTop = messagesEl.scrollHeight;
}

function sendPayload(payload) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return false;
    socket.send(JSON.stringify(payload));
    return true;
}

function sendText() {
    const content = messageInput.value.trim().slice(0, maxMessageLength);
    if (!content) return;

    const hasOnlyEmoji = /^\p{Extended_Pictographic}+$/u.test(content.replace(/\s/g, ''));
    if (!sendPayload({ type: hasOnlyEmoji ? 'emoji' : 'chat', content })) return;
    messageInput.value = '';
    updateCharCount();
    messageInput.focus();
}

function sendSticker(stickerUrl) {
    sendPayload({ type: 'sticker', content: '发送了一个表情包', stickerUrl });
    emojiPanel.hidden = true;
    messageInput.focus();
}

function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

    socket.onopen = () => {
        setConnected(true);
    };

    socket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        setOnlineCount(data.onlineCount);
        if (data.type !== 'online') addMessage(data);
    };

    socket.onclose = () => {
        setConnected(false);
        setOnlineCount(0);
        addMessage({ type: 'system', username: '系统', content: 'WebSocket 连接已关闭', time: new Date().toLocaleTimeString() });
    };

    socket.onerror = () => {
        addMessage({ type: 'system', username: '系统', content: 'WebSocket 发生错误', time: new Date().toLocaleTimeString() });
    };
}

async function loadCurrentUser() {
    const response = await fetch('/api/me');
    if (!response.ok) {
        window.location.href = '/login';
        return;
    }

    const data = await response.json();
    currentUser = data.username;
    currentUserEl.textContent = currentUser;
    connectWebSocket();
}

function renderStickers() {
    stickers.forEach((stickerUrl) => {
        const button = document.createElement('button');
        button.type = 'button';
        const img = document.createElement('img');
        img.src = stickerUrl;
        img.alt = '表情包';
        button.appendChild(img);
        button.addEventListener('click', () => sendSticker(stickerUrl));
        stickerList.appendChild(button);
    });
}

document.querySelectorAll('.emoji-list button').forEach((button) => {
    button.addEventListener('click', () => {
        messageInput.value += button.textContent;
        messageInput.focus();
    });
});

emojiBtn.addEventListener('click', () => {
    emojiPanel.hidden = !emojiPanel.hidden;
});

document.addEventListener('click', (event) => {
    if (emojiPanel.hidden) return;
    if (emojiPanel.contains(event.target) || emojiBtn.contains(event.target)) return;
    emojiPanel.hidden = true;
});

document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') emojiPanel.hidden = true;
});

sendBtn.addEventListener('click', sendText);
messageInput.addEventListener('input', updateCharCount);
messageInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') {
        event.preventDefault();
        sendText();
    }
});

logoutForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    await fetch('/api/logout', { method: 'POST' });
    window.location.href = '/login';
});

setConnected(false);
setOnlineCount(0);
messageInput.maxLength = maxMessageLength;
updateCharCount();
renderStickers();
loadCurrentUser();
