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
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;
const baseReconnectDelay = 1000;
let reconnectTimer = null;

function setConnected(connected) {
    statusEl.textContent = connected ? '已连接' : '未连接';
    statusEl.classList.toggle('connected', connected);
    statusEl.classList.toggle('disconnected', !connected);
    statusEl.classList.remove('reconnecting');
    messageInput.disabled = !connected;
    emojiBtn.disabled = !connected;
    sendBtn.disabled = !connected;
}

function setReconnecting() {
    statusEl.textContent = '重连中...';
    statusEl.classList.remove('connected');
    statusEl.classList.add('disconnected', 'reconnecting');
    messageInput.disabled = true;
    emojiBtn.disabled = true;
    sendBtn.disabled = true;
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
    if (data.type === 'error') item.classList.add('error-message');
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
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        addMessage({
            type: 'error',
            username: '系统',
            content: '消息发送失败：连接已断开，请等待重连或刷新页面',
            time: new Date().toLocaleTimeString()
        });
        return false;
    }
    try {
        socket.send(JSON.stringify(payload));
        return true;
    } catch (err) {
        addMessage({
            type: 'error',
            username: '系统',
            content: '消息发送失败：网络异常',
            time: new Date().toLocaleTimeString()
        });
        return false;
    }
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

function getReconnectDelay() {
    return Math.min(baseReconnectDelay * Math.pow(2, reconnectAttempts), 30000);
}

function connectWebSocket() {
    if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

    socket.onopen = () => {
        reconnectAttempts = 0;
        setConnected(true);
        addMessage({
            type: 'system',
            username: '系统',
            content: 'WebSocket 连接已建立',
            time: new Date().toLocaleTimeString()
        });
    };

    socket.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            setOnlineCount(data.onlineCount);
            if (data.type !== 'online') addMessage(data);
        } catch (err) {
            console.error('Failed to parse message:', err);
        }
    };

    socket.onclose = (event) => {
        setConnected(false);
        setOnlineCount(0);

        if (reconnectAttempts < maxReconnectAttempts) {
            setReconnecting();
            const delay = getReconnectDelay();
            reconnectAttempts++;
            addMessage({
                type: 'system',
                username: '系统',
                content: `连接已断开，${Math.round(delay / 1000)}秒后尝试第 ${reconnectAttempts}/${maxReconnectAttempts} 次重连...`,
                time: new Date().toLocaleTimeString()
            });
            reconnectTimer = setTimeout(connectWebSocket, delay);
        } else {
            addMessage({
                type: 'error',
                username: '系统',
                content: '重连失败，请刷新页面重试',
                time: new Date().toLocaleTimeString()
            });
        }
    };

    socket.onerror = () => {
        console.error('WebSocket error');
    };
}

async function loadCurrentUser() {
    try {
        const response = await fetch('/api/me');
        if (!response.ok) {
            window.location.href = '/login';
            return;
        }

        const data = await response.json();
        currentUser = data.username;
        currentUserEl.textContent = currentUser;
        connectWebSocket();
    } catch (err) {
        console.error('Failed to load user:', err);
        window.location.href = '/login';
    }
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
    reconnectAttempts = maxReconnectAttempts;
    if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
    }
    if (socket) {
        socket.onclose = null;
        socket.close();
        socket = null;
    }
    await fetch('/api/logout', { method: 'POST' });
    window.location.href = '/login';
});

setConnected(false);
setOnlineCount(0);
messageInput.maxLength = maxMessageLength;
updateCharCount();
renderStickers();
loadCurrentUser();
