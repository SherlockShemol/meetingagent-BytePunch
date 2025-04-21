// Global state
let currentMeetingId = null;
let currentSessionId = null;
let currentMeetingContent = null;
let jsonEditor = null;
let summaryJsonEditor = null;

// DOM Elements
const meetingList = document.getElementById('meetingList');
const createMeetingBtn = document.getElementById('createMeetingBtn');
const fileInput = document.getElementById('fileInput');
const noMeetingSelected = document.getElementById('noMeetingSelected');
const meetingDetails = document.getElementById('meetingDetails');
const contentViewer = document.getElementById('contentViewer');
const summaryJsonViewer = document.getElementById('summaryJsonViewer');
const summaryMarkdown = document.getElementById('summaryMarkdown');
const jsonPathInput = document.getElementById('jsonPathInput');
const convertToMarkdownBtn = document.getElementById('convertToMarkdownBtn');
const showJsonBtn = document.getElementById('showJsonBtn');
const chatMessages = document.getElementById('chatMessages');
const chatInput = document.getElementById('chatInput');
const sendMessageBtn = document.getElementById('sendMessageBtn');
const taskList = document.getElementById('taskList'); // 新增 Task 列表元素

// URL State Management
function updateURLState() {
  const params = new URLSearchParams();
  if (currentMeetingId) params.set('meeting', currentMeetingId);
  if (currentTab) params.set('tab', currentTab);
  // 注意：如果需要在 URL 中保留 JSON Path，需要添加逻辑
  // if (currentTab === 'summary' && currentJsonPath) params.set('path', currentJsonPath);

  const newURL = `${window.location.pathname}?${params.toString()}`;
  window.history.pushState({}, '', newURL);
}

function loadURLState() {
  const params = new URLSearchParams(window.location.search);
  const meetingId = params.get('meeting');
  const tab = params.get('tab') || 'content'; // 默认是 content
  const path = params.get('path'); // 暂时保留，虽然没在 Task 中使用

  // 先切换 Tab 再加载 Meeting，确保 Meeting 加载时 Tab 状态正确
  if (tab) {
    switchTab(tab);
  }

  if (meetingId) {
    selectMeeting(meetingId); // selectMeeting 会处理加载数据和更新 UI
  }

  // 如果 URL 指定了 Summary Tab 和 Path，需要特殊处理
  // if (tab === 'summary' && path && summaryJsonEditor) {
  //   jsonPathInput.value = path;
  //   convertToMarkdown(); // 假设 convertToMarkdown 会处理显示
  // }
}

// Tab Management
let currentTab = 'content'; // 确保默认值存在
let currentJsonPath = '';

function switchTab(tab) {
  currentTab = tab;
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.tab === tab);
  });
  document.querySelectorAll('.tab-content').forEach(content => {
    // 确保 content.id 匹配 HTML 中的 ID (e.g., 'contentTab', 'summaryTab', 'chatTab', 'taskTab')
    content.classList.toggle('active', content.id === `${tab}Tab`);
  });
  updateURLState();

  // 如果切换到 Task Tab 且有选中的会议，则加载任务
  if (tab === 'task' && currentMeetingId) {
    loadTasks(currentMeetingId);
  }
}

// Initialize JSON Editor
function initJsonEditor() {
  const options = {
    mode: 'view',
    modes: ['view', 'code'],
    onModeChange: function (newMode) {
      if (newMode === 'code') {
        jsonEditor.expandAll();
      }
    }
  };
  jsonEditor = new JSONEditor(contentViewer, options);
}

// Initialize Summary JSON Editor
function initSummaryJsonEditor() {
  const options = {
    mode: 'view',
    modes: ['view', 'code'],
    onModeChange: function (newMode) {
      if (newMode === 'code') {
        summaryJsonEditor.expandAll();
      }
    }
  };
  summaryJsonEditor = new JSONEditor(summaryJsonViewer, options);
}

// Get value by JSON path
function getValueByPath(obj, path) {
  const parts = path.split('.');
  let current = obj;

  for (const part of parts) {
    if (part === '$') continue;
    if (current === undefined || current === null) return null;
    current = current[part];
  }

  return current;
}

// Event Listeners
createMeetingBtn.addEventListener('click', () => fileInput.click());
fileInput.addEventListener('change', handleFileUpload);
sendMessageBtn.addEventListener('click', sendMessage);
chatInput.addEventListener('keypress', (e) => {
  if (e.key === 'Enter') sendMessage();
});

convertToMarkdownBtn.addEventListener('click', convertToMarkdown);
showJsonBtn.addEventListener('click', showJson);

// Tab switching
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    switchTab(btn.dataset.tab);
  });
});

// Functions
function convertToMarkdown() {
  const path = jsonPathInput.value.trim();
  if (!path) return;

  try {
    const summaryData = summaryJsonEditor.get();
    const value = getValueByPath(summaryData, path);

    if (value === undefined || value === null) {
      alert('No value found at the specified path');
      return;
    }

    // Show raw content
    const content = typeof value === 'string' ? value : JSON.stringify(value, null, 2);

    // Show markdown
    summaryJsonViewer.classList.add('hidden');
    summaryMarkdown.classList.remove('hidden');
    summaryMarkdown.textContent = content;

    // Update URL state
    currentJsonPath = path;
    updateURLState();
  } catch (error) {
    console.error('Error:', error);
    alert('Error converting to markdown');
  }
}

function showJson() {
  summaryJsonViewer.classList.remove('hidden');
  summaryMarkdown.classList.add('hidden');
  currentJsonPath = '';
  updateURLState();
}

async function handleFileUpload(e) {
  const file = e.target.files[0];
  if (!file) return;

  try {
    const content = await file.text();
    const response = await fetch('/meeting', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: content
    });

    if (!response.ok) throw new Error('Failed to create meeting');

    const data = await response.json();
    loadMeetings();
    selectMeeting(data.id);
  } catch (error) {
    console.error('Error:', error);
    alert('Failed to create meeting');
  }
}

async function loadMeetings() {
  try {
    const response = await fetch('/meeting');
    const data = await response.json();

    meetingList.innerHTML = data.meetings.map(meeting => `
            <div class="meeting-item" data-id="${meeting.id}">
                <div class="font-medium">${meeting.content.title || 'Untitled Meeting'}</div>
                <div class="text-sm text-gray-500">${new Date().toLocaleDateString()}</div>
            </div>
        `).join('');

    // Add click handlers to meeting items
    document.querySelectorAll('.meeting-item').forEach(item => {
      item.addEventListener('click', () => selectMeeting(item.dataset.id));
    });
  } catch (error) {
    console.error('Error:', error);
  }
}

// 新增: 加载任务列表的函数
async function loadTasks(meetingId) {
  if (!meetingId) return;
  taskList.innerHTML = '<p class="text-gray-500">Loading tasks...</p>'; // 显示加载状态

  try {
    // 确保这里调用的是 /tasks 接口
    const response = await fetch(`/tasks?meeting_id=${meetingId}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch tasks: ${response.statusText}`);
    }
    const tasks = await response.json(); // 假设返回 { tasks: ["task1", "task2", ...] }

    if (tasks && tasks.tasks && tasks.tasks.length > 0) {
      // 根据实际返回的数据结构调整渲染逻辑
      taskList.innerHTML = tasks.tasks.map((task, index) => `
        <div class="p-2 border-b">
          <input type="checkbox" id="task-${index}" class="mr-2">
          <label for="task-${index}">${typeof task === 'string' ? task : JSON.stringify(task)}</label>
        </div>
      `).join('');
    } else {
      taskList.innerHTML = '<p class="text-gray-500">No tasks found for this meeting.</p>';
    }
  } catch (error) {
    console.error('Error loading tasks:', error);
    taskList.innerHTML = '<p class="text-red-500">Failed to load tasks.</p>';
  }
}


async function selectMeeting(meetingId) {
  currentMeetingId = meetingId;
  currentSessionId = `session_${Date.now()}`;

  // Update UI
  document.querySelectorAll('.meeting-item').forEach(item => {
    item.classList.toggle('active', item.dataset.id === meetingId);
  });

  noMeetingSelected.classList.add('hidden');
  meetingDetails.classList.remove('hidden');

  // Load meeting content
  try {
    const response = await fetch('/meeting');
    const data = await response.json();
    const meeting = data.meetings.find(m => m.id === meetingId);
    if (meeting) {
      currentMeetingContent = meeting.content;
      // Update JSON editor
      if (!jsonEditor) {
        initJsonEditor();
      }
      jsonEditor.set(meeting.content);
      jsonEditor.expandAll();
    }
  } catch (error) {
    console.error('Error loading meeting content:', error);
  }

  // Load summary
  try {
    const response = await fetch(`/summary?meeting_id=${meetingId}`);
    const data = await response.json();

    // Initialize summary JSON editor if not exists
    if (!summaryJsonEditor) {
      initSummaryJsonEditor();
    }

    // Update summary JSON editor
    summaryJsonEditor.set(data);
    summaryJsonEditor.expandAll();

    // Reset markdown view
    summaryJsonViewer.classList.remove('hidden');
    summaryMarkdown.classList.add('hidden');
    jsonPathInput.value = '';
    currentJsonPath = ''; // 重置 JSON Path 状态

  } catch (error) {
    console.error('Error loading summary:', error);
  }

  // Clear chat
  chatMessages.innerHTML = '';
  msgs = {}; // 清空消息缓存

  // Load tasks if Task tab is active or becomes active later via switchTab
  if (currentTab === 'task') {
      loadTasks(meetingId);
  } else {
      // Optionally clear task list if not the active tab, or leave as is
      taskList.innerHTML = '<p class="text-gray-500">Select the Tasks tab to view.</p>';
  }


  // Update URL state *after* all data loading attempts and UI updates
  updateURLState();
}

async function sendMessage() {
  const message = chatInput.value.trim();
  if (!message || !currentMeetingId || !currentSessionId) return;

  // Add user message to chat
  const userMsgID = Math.random().toString(36).substring(2, 15);
  addMessageToChat(userMsgID, message, 'user');
  chatInput.value = '';

  // Start SSE connection and send message
  const eventSource = new EventSource(`/chat?meeting_id=${currentMeetingId}&session_id=${currentSessionId}&message=${encodeURIComponent(message)}`);
  const assistantMsgID = Math.random().toString(36).substring(2, 15);

  eventSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    // 从响应中提取消息内容
    addMessageToChat(assistantMsgID, data.data.message, 'assistant');
  };

  eventSource.onerror = () => {
    eventSource.close();
  };
}

let msgs = {};

function addMessageToChat(msgID, message, type) {
  if (msgs[msgID]) {
    msgs[msgID].textContent += message;
  } else {
    const messageDiv = document.createElement('div');
    messageDiv.className = `chat-message ${type}`;
    messageDiv.textContent = message;
    chatMessages.appendChild(messageDiv);
    msgs[msgID] = messageDiv;
  }

  chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Initialize
loadMeetings();
loadURLState(); // loadURLState 会处理初始 Tab 和 Meeting 的加载