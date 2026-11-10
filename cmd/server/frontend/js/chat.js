// AI chat sidebar. Rendered into #chat-root, outside #app, so the app's
// render() can rebuild the page without disturbing the conversation.

const CHAT_STORAGE_KEY = 'chat';
const CHAT_MAX_IMAGES = 3;
const CHAT_IMAGE_MAX_DIMENSION = 1600;
const CHAT_HISTORY_LIMIT = 30;
const CHAT_MESSAGE_MAX_CHARS = 8000;
const CHAT_INPUT_MAX_HEIGHT = 200; // px; keep in sync with .chat-input max-height

// A message is { role: 'user' | 'assistant', content, images?, imageCount?,
// actions?, error? }. Image data lives in memory only; sessionStorage keeps
// just the count.
const chat = {
    open: false,
    userId: null,
    conversationId: '',
    messages: [],
    pendingImages: [],
    busy: false,
    status: null,
    abortController: null
};

function chatNewConversationId() {
    if (window.crypto && typeof crypto.randomUUID === 'function') {
        return crypto.randomUUID();
    }
    return Date.now().toString(36) + Math.random().toString(36).slice(2, 12);
}

function chatSave() {
    try {
        sessionStorage.setItem(CHAT_STORAGE_KEY, JSON.stringify({
            open: chat.open,
            userId: chat.userId,
            conversationId: chat.conversationId,
            messages: chat.messages.map(({ images, ...rest }) => rest)
        }));
    } catch (error) {
        console.error('Failed to save chat:', error);
    }
}

function chatLoad() {
    try {
        const saved = JSON.parse(sessionStorage.getItem(CHAT_STORAGE_KEY) || 'null');
        if (saved && Array.isArray(saved.messages)) {
            chat.open = !!saved.open;
            chat.userId = saved.userId ?? null;
            chat.conversationId = saved.conversationId || '';
            chat.messages = saved.messages;
        }
    } catch (error) {
        console.error('Failed to load chat:', error);
    }
}

// Drops the conversation, e.g. on logout or when starting a new chat.
function chatReset({ keepOpen = false } = {}) {
    if (chat.abortController) {
        chat.abortController.abort();
    }
    chat.open = keepOpen && chat.open;
    chat.userId = state.user ? state.user.id : null;
    chat.conversationId = chatNewConversationId();
    chat.messages = [];
    chat.pendingImages = [];
    chat.busy = false;
    chat.status = null;
    chatSave();
}

function chatStartNew() {
    chatReset({ keepOpen: true });
    renderChat();
    document.getElementById('chat-input')?.focus();
}

function toggleChat() {
    chat.open = !chat.open;
    chatSave();
    renderChat();
    if (chat.open) {
        document.getElementById('chat-input')?.focus();
    }
}

function renderChat() {
    const root = document.getElementById('chat-root');
    if (!root) return;

    if (!state.aiChatEnabled || !state.user) {
        root.innerHTML = '';
        document.body.classList.remove('chat-open');
        return;
    }

    // A conversation never carries over to a different user.
    if (chat.userId !== state.user.id || !chat.conversationId) {
        chatReset({ keepOpen: chat.userId === null });
    }

    if (!chat.open) {
        root.innerHTML = '';
        document.body.classList.remove('chat-open');
        return;
    }

    // Keep whatever the user is typing across re-renders.
    const previousInput = document.getElementById('chat-input');
    const draft = previousInput ? previousInput.value : '';
    const hadFocus = previousInput && document.activeElement === previousInput;

    root.innerHTML = `
        <aside class="chat-panel" aria-label="${escapeHTMLAttr(t('chat.title'))}">
            <div class="chat-header">
                <h2>${t('chat.title')}</h2>
                <div class="chat-header-actions">
                    <button class="chat-text-btn" onclick="chatStartNew()">${t('chat.new')}</button>
                    <button class="chat-close-btn" onclick="toggleChat()" aria-label="${escapeHTMLAttr(t('chat.close'))}">&times;</button>
                </div>
            </div>
            <div id="chat-messages" class="chat-messages"></div>
            <div id="chat-attachments" class="chat-attachments"></div>
            <form class="chat-form" onsubmit="chatSubmit(event)">
                <input type="file" id="chat-file-input" accept="image/*" multiple hidden onchange="chatAddFiles(this)">
                <input type="file" id="chat-camera-input" accept="image/*" capture="environment" hidden onchange="chatAddFiles(this)">
                <textarea id="chat-input" class="input chat-input" rows="2" maxlength="${CHAT_MESSAGE_MAX_CHARS}"
                          placeholder="${escapeHTMLAttr(t('chat.placeholder'))}"
                          onkeydown="chatInputKeydown(event)" oninput="chatAutoGrow(this)"></textarea>
                <div class="chat-form-actions">
                    <button type="button" class="chat-icon-btn chat-camera-btn" onclick="document.getElementById('chat-camera-input').click()"
                            aria-label="${escapeHTMLAttr(t('chat.camera'))}" title="${escapeHTMLAttr(t('chat.camera'))}">📷</button>
                    <button type="button" class="chat-icon-btn" onclick="document.getElementById('chat-file-input').click()"
                            aria-label="${escapeHTMLAttr(t('chat.attach'))}" title="${escapeHTMLAttr(t('chat.attach'))}">📎</button>
                    <button type="submit" id="chat-send-btn" class="btn btn-primary chat-send-btn"></button>
                </div>
            </form>
        </aside>
    `;
    document.body.classList.add('chat-open');

    const input = document.getElementById('chat-input');
    input.value = draft;
    chatAutoGrow(input);
    if (hadFocus) {
        input.focus();
    }

    renderChatAttachments();
    renderChatMessages(true);
}

function renderChatMessages(forceScroll = false) {
    const list = document.getElementById('chat-messages');
    if (!list) return;

    const nearBottom = list.scrollHeight - list.scrollTop - list.clientHeight < 80;

    if (chat.messages.length === 0) {
        list.innerHTML = `
            <div class="chat-empty">
                <p>${t('chat.empty')}</p>
                ${['chat.example1', 'chat.example2', 'chat.example3'].map(key => `
                    <button class="chat-example" onclick="chatSend(t('${key}'))">${t(key)}</button>
                `).join('')}
            </div>
        `;
    } else {
        list.innerHTML = chat.messages.map(ChatMessage).join('');
    }

    const sendButton = document.getElementById('chat-send-btn');
    if (sendButton) {
        sendButton.textContent = chat.busy ? t('chat.stop') : t('chat.send');
    }

    if (forceScroll || nearBottom) {
        list.scrollTop = list.scrollHeight;
    }
}

function ChatMessage(message, index) {
    if (message.role === 'user') {
        let images = '';
        if (message.images && message.images.length > 0) {
            images = `<div class="chat-message-images">${message.images.map(src =>
                `<img src="${escapeHTMLAttr(src)}" alt="">`).join('')}</div>`;
        } else if (message.imageCount) {
            images = `<div class="chat-image-note">📷 ${t('chat.image_placeholder')}${message.imageCount > 1 ? ` ×${message.imageCount}` : ''}</div>`;
        }
        return `
            <div class="chat-message chat-message-user">
                ${images}
                ${message.content ? `<div class="chat-bubble">${escapeHTML(message.content)}</div>` : ''}
            </div>
        `;
    }

    const isLast = index === chat.messages.length - 1;
    const waiting = isLast && chat.busy;
    let status = '';
    if (waiting && (!message.content || chat.status)) {
        status = `<div class="chat-status">${chat.status ? t(chat.status) : t('chat.thinking')}</div>`;
    }

    return `
        <div class="chat-message chat-message-assistant">
            ${message.content ? `<div class="chat-bubble">${chatFormat(message.content)}</div>` : ''}
            ${(message.actions || []).map((action, actionIndex) => ChatActionCard(action, index, actionIndex)).join('')}
            ${status}
            ${message.error ? `<div class="chat-error">${t(message.error)}</div>` : ''}
        </div>
    `;
}

function ChatActionCard(action, messageIndex, actionIndex) {
    const category = action.category_id ? state.categories.find(c => c.id === action.category_id) : null;
    const isIncome = action.type === 'income';
    return `
        <div class="chat-action-card ${action.undone ? 'chat-action-undone' : ''}">
            <div class="chat-action-main">
                <div class="chat-action-title">${escapeHTML(action.description)}</div>
                <div class="chat-action-meta">
                    ${formatDateForDisplay(action.date)}${category ? ` · ${escapeHTML(category.description)}` : ''}
                </div>
            </div>
            <div class="chat-action-side">
                <div class="${isIncome ? 'amount-success' : 'amount-danger'}">
                    ${isIncome ? '+' : '-'}${escapeHTML(state.currency)}${Number(action.amount).toFixed(2)}
                </div>
                ${action.undone
                    ? `<span class="chat-action-meta">${t('chat.undone')}</span>`
                    : `<button class="chat-text-btn" onclick="chatUndoAction(${messageIndex}, ${actionIndex})">${t('chat.undo')}</button>`}
            </div>
        </div>
    `;
}

// Minimal, safe formatting of model output: escaped text with **bold** and
// "- " list markers. Line breaks are kept by CSS (white-space: pre-wrap).
function chatFormat(text) {
    return escapeHTML(text)
        .replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>')
        .replace(/^(\s*)[-*] /gm, '$1• ');
}

function renderChatAttachments() {
    const container = document.getElementById('chat-attachments');
    if (!container) return;
    container.innerHTML = chat.pendingImages.map((src, index) => `
        <div class="chat-attachment">
            <img src="${escapeHTMLAttr(src)}" alt="">
            <button type="button" onclick="chatRemoveImage(${index})" aria-label="${escapeHTMLAttr(t('chat.remove_image'))}">&times;</button>
        </div>
    `).join('');
}

function chatRemoveImage(index) {
    chat.pendingImages.splice(index, 1);
    renderChatAttachments();
}

// Downscales a photo to a JPEG data URL: receipts stay readable while the
// upload stays small.
async function chatPrepareImage(file) {
    const bitmap = await createImageBitmap(file);
    const scale = Math.min(1, CHAT_IMAGE_MAX_DIMENSION / Math.max(bitmap.width, bitmap.height));
    const canvas = document.createElement('canvas');
    canvas.width = Math.max(1, Math.round(bitmap.width * scale));
    canvas.height = Math.max(1, Math.round(bitmap.height * scale));
    const context = canvas.getContext('2d');
    // JPEG has no alpha channel; avoid transparent areas turning black.
    context.fillStyle = '#ffffff';
    context.fillRect(0, 0, canvas.width, canvas.height);
    context.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
    if (typeof bitmap.close === 'function') {
        bitmap.close();
    }
    return canvas.toDataURL('image/jpeg', 0.85);
}

async function chatAddFiles(input) {
    const files = Array.from(input.files || []);
    input.value = '';

    for (const file of files) {
        if (chat.pendingImages.length >= CHAT_MAX_IMAGES) {
            alert(t('chat.images_limit'));
            break;
        }
        try {
            chat.pendingImages.push(await chatPrepareImage(file));
        } catch (error) {
            console.error('Failed to read image:', error);
            alert(t('chat.image_error'));
        }
    }
    renderChatAttachments();
    document.getElementById('chat-input')?.focus();
}

function chatAutoGrow(textarea) {
    // scrollHeight excludes the 1px borders of the border-box textarea.
    textarea.style.height = 'auto';
    const height = textarea.scrollHeight + 2;
    textarea.style.height = Math.min(height, CHAT_INPUT_MAX_HEIGHT) + 'px';
    textarea.style.overflowY = height > CHAT_INPUT_MAX_HEIGHT ? 'auto' : 'hidden';
}

function chatInputKeydown(event) {
    if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
        event.preventDefault();
        chatSubmit(event);
    }
}

function chatSubmit(event) {
    event.preventDefault();
    if (chat.busy) {
        if (event.type === 'submit' && chat.abortController) {
            chat.abortController.abort();
        }
        return;
    }
    const input = document.getElementById('chat-input');
    const text = input.value.trim();
    if (!text && chat.pendingImages.length === 0) return;
    input.value = '';
    chatAutoGrow(input);
    chatSend(text);
}

// Conversation as sent to the server: text only, except for the images of
// the message being sent. Entries the assistant created are noted in its
// messages so later turns know about them.
function chatHistoryForServer() {
    const lastIndex = chat.messages.length - 1;
    const history = [];

    chat.messages.forEach((message, index) => {
        if (message.role === 'user') {
            const item = { role: 'user', content: message.content || '' };
            if (index === lastIndex && message.images && message.images.length > 0) {
                item.images = message.images;
            } else if (message.imageCount) {
                item.content = `${item.content}\n[${message.imageCount} image(s) attached earlier, no longer available]`.trim();
            }
            if (item.content || item.images) {
                history.push(item);
            }
            return;
        }

        let content = message.content || '';
        (message.actions || []).forEach(action => {
            content += `\n[Created entry #${action.id}: ${action.type} ${Number(action.amount).toFixed(2)} on ${action.date}, "${action.description}"${action.undone ? ' (later removed by the user)' : ''}]`;
        });
        content = content.trim();
        if (content) {
            history.push({ role: 'assistant', content });
        }
    });

    return history.slice(-CHAT_HISTORY_LIMIT).map(item => ({
        ...item,
        content: item.content.slice(0, CHAT_MESSAGE_MAX_CHARS)
    }));
}

async function chatSend(text) {
    if (chat.busy) return;

    const images = chat.pendingImages;
    chat.pendingImages = [];
    chat.messages.push({ role: 'user', content: text, images, imageCount: images.length });

    const history = chatHistoryForServer();
    const reply = { role: 'assistant', content: '', actions: [] };
    chat.messages.push(reply);

    chat.busy = true;
    chat.status = null;
    chat.abortController = new AbortController();
    renderChatAttachments();
    renderChatMessages(true);

    try {
        const response = await fetch('/api/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'same-origin',
            signal: chat.abortController.signal,
            body: JSON.stringify({
                conversation_id: chat.conversationId,
                language: state.language,
                messages: history
            })
        });

        if (response.status === 401) {
            state.user = null;
            render();
            return;
        }
        if (!response.ok || !response.body) {
            throw new Error(`chat request failed with status ${response.status}`);
        }

        await chatReadStream(response.body, event => chatHandleEvent(reply, event));
    } catch (error) {
        if (error.name !== 'AbortError') {
            console.error('Chat error:', error);
            reply.error = 'chat.error';
        }
    } finally {
        chat.busy = false;
        chat.status = null;
        chat.abortController = null;
        chatSave();
        renderChatMessages();
    }
}

// Reads a server-sent event stream, calling onEvent with each decoded
// "data:" payload.
async function chatReadStream(body, onEvent) {
    const reader = body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        let boundary;
        while ((boundary = buffer.indexOf('\n\n')) !== -1) {
            const block = buffer.slice(0, boundary);
            buffer = buffer.slice(boundary + 2);
            for (const line of block.split('\n')) {
                if (line.startsWith('data:')) {
                    onEvent(JSON.parse(line.slice(5)));
                }
            }
        }
    }
}

function chatHandleEvent(reply, event) {
    switch (event.type) {
        case 'delta':
            reply.content += event.text;
            chat.status = null;
            break;
        case 'tool':
            chat.status = event.name === 'create_action' ? 'chat.status_creating' : 'chat.status_reading';
            break;
        case 'action':
            reply.actions.push(event.action);
            chatRefreshAppData();
            break;
        case 'error':
            reply.error = event.message === 'too_many_steps' ? 'chat.error_steps' : 'chat.error';
            break;
    }
    renderChatMessages();
}

// Reloads whatever the current page shows after the assistant changed data.
function chatRefreshAppData() {
    if (state.currentPage === 'all-actions') {
        loadAllActions();
    } else if (state.currentPage === 'charts') {
        loadChartData();
        loadCategoryChartData();
    } else if (state.currentPage === 'dashboard') {
        loadActions();
    }
}

async function chatUndoAction(messageIndex, actionIndex) {
    const action = chat.messages[messageIndex]?.actions?.[actionIndex];
    if (!action || action.undone) return;

    const data = await api(`/api/actions/${action.id}`, { method: 'DELETE' });
    if (!data) {
        alert(t('chat.undo_failed'));
        return;
    }
    action.undone = true;
    chatSave();
    renderChatMessages();
    chatRefreshAppData();
}

chatLoad();
