(function () {
  var state = {
    open: false,
    sending: false,
    conversationId: localStorage.getItem('ai_conversation_id') || ''
  };

  function el(tag, className, text) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (text) node.textContent = text;
    return node;
  }

  function createAssistant() {
    if (document.getElementById('aiAssistant')) return;

    var root = el('div', 'ai-assistant', '');
    root.id = 'aiAssistant';

    var panel = el('div', 'ai-panel', '');
    panel.innerHTML = '' +
      '<div class="ai-header">' +
      '  <div><strong>Future AI</strong><span>站内智能助手</span></div>' +
      '  <button class="ai-close" type="button" aria-label="关闭">×</button>' +
      '</div>' +
      '<div class="ai-messages" id="aiMessages"></div>' +
      '<div class="ai-suggestions">' +
      '  <button type="button">这个网站怎么用？</button>' +
      '  <button type="button">帮我生成个人简介</button>' +
      '  <button type="button">未来实验室是什么？</button>' +
      '</div>' +
      '<form class="ai-input-row" id="aiForm">' +
      '  <textarea id="aiInput" rows="1" maxlength="1000" placeholder="问我网站使用、账号问题、文案生成..."></textarea>' +
      '  <button type="submit">发送</button>' +
      '</form>';

    var bubble = el('button', 'ai-bubble', '');
    bubble.type = 'button';
    bubble.innerHTML = '<span>AI</span><small>助手</small>';

    root.appendChild(panel);
    root.appendChild(bubble);
    document.body.appendChild(root);

    addMessage('assistant', '你好，我是 Future AI。可以帮你了解网站功能、账号操作，也能帮你写简介、标题和文案。');

    bubble.addEventListener('click', function () { togglePanel(true); });
    panel.querySelector('.ai-close').addEventListener('click', function () { togglePanel(false); });
    panel.querySelectorAll('.ai-suggestions button').forEach(function (btn) {
      btn.addEventListener('click', function () {
        sendMessage(btn.textContent);
      });
    });
    document.getElementById('aiForm').addEventListener('submit', function (e) {
      e.preventDefault();
      var input = document.getElementById('aiInput');
      sendMessage(input.value);
    });
  }

  function togglePanel(open) {
    state.open = open;
    var root = document.getElementById('aiAssistant');
    if (!root) return;
    root.classList.toggle('open', open);
    if (open) setTimeout(function () { document.getElementById('aiInput').focus(); }, 120);
  }

  function addMessage(role, text, extraClass) {
    var box = document.getElementById('aiMessages');
    if (!box) return;
    var item = el('div', 'ai-msg ' + role + (extraClass ? ' ' + extraClass : ''), '');
    item.textContent = text;
    box.appendChild(item);
    box.scrollTop = box.scrollHeight;
    return item;
  }

  async function sendMessage(raw) {
    var message = (raw || '').trim();
    if (!message || state.sending) return;
    togglePanel(true);
    state.sending = true;

    var input = document.getElementById('aiInput');
    if (input) input.value = '';

    addMessage('user', message);
    var loading = addMessage('assistant', '思考中...', 'loading');

    try {
      var res = await fetch('/api/ai/chat', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: message, conversationId: state.conversationId })
      });
      var data = await res.json();
      if (loading) loading.remove();
      if (data.code === 0 && data.data) {
        state.conversationId = data.data.conversationId || state.conversationId;
        if (state.conversationId) localStorage.setItem('ai_conversation_id', state.conversationId);
        addMessage('assistant', data.data.reply + (data.data.mock ? '\n\n（提示：当前是本地体验模式，配置 AI_API_KEY 后会调用真实大模型。）' : ''));
      } else {
        addMessage('assistant', data.msg || 'AI 暂时没有响应，请稍后再试。');
      }
    } catch (err) {
      if (loading) loading.remove();
      addMessage('assistant', '网络开小差了，请稍后再试。');
    } finally {
      state.sending = false;
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', createAssistant);
  } else {
    createAssistant();
  }
})();
