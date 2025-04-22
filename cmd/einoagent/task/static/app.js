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
                <div class="font-medium">${meeting.id || 'Untitled Meeting'}</div>
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
    console.log('Received summary data:', data);
    const summaryContent = data && typeof data.summary === 'string' ? data.summary : '无摘要内容';

    // 显示 Markdown 内容
    summaryMarkdown.classList.remove('hidden');
    summaryJsonViewer.classList.add('hidden');
    summaryMarkdown.innerHTML = marked.parse(summaryContent); // 渲染 Markdown

  } catch (error) {
    console.error('Error loading summary:', error);
    summaryMarkdown.classList.remove('hidden');
    summaryMarkdown.textContent = '无法加载摘要内容';
  }
  //   const summaryContent = data.summary || '';
  //   summaryJsonEditor.set({ summary: summaryContent });
  //   summaryJsonEditor.expandAll();

  //   // Reset markdown view
  //   summaryJsonViewer.classList.remove('hidden');
  //   summaryMarkdown.classList.add('hidden');
  //   jsonPathInput.value = '';
  //   currentJsonPath = ''; // 重置 JSON Path 状态

  // } catch (error) {
  //   console.error('Error loading summary:', error);
  // }

  // Clear chat
  chatMessages.innerHTML = '';
  msgs = {}; // 清空消息缓存

  // Load tasks if Task tab is active or becomes active later via switchTab
  if (currentTab === 'task') {
      loadTasks(meetingId);
  } else {
      // Optionally clear task list if not the active tab, or leave as is
      // 移除之前的提示，让 loadTasks 自己处理初始状态或错误状态
      // taskList.innerHTML = '<p class="text-gray-500">Select the Tasks tab to view.</p>';
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
  const basePath = window.location.pathname.startsWith('/task') ? '/..' : '';
  const eventSource = new EventSource(`${basePath}/chat?meeting_id=${currentMeetingId}&session_id=${currentSessionId}&message=${encodeURIComponent(message)}`);
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



// URL 参数处理
function getQueryParams() {
    const params = new URLSearchParams(window.location.search);
    return {
        query: params.get('q') || '',
        is_done: params.get('done') === null ? null : params.get('done') === 'true',
        limit: parseInt(params.get('limit')) || 10,
        sort: params.get('sort') || 'urgency'
    };
}

function updateQueryParams(params) {
    const url = new URL(window.location);
    if (params.query) url.searchParams.set('q', params.query);
    else url.searchParams.delete('q');

    if (params.is_done !== null) url.searchParams.set('done', params.is_done);
    else url.searchParams.delete('done');

    if (params.limit !== 10) url.searchParams.set('limit', params.limit);
    else url.searchParams.delete('limit');

    if (params.sort !== 'urgency') url.searchParams.set('sort', params.sort);
    else url.searchParams.delete('sort');

    window.history.pushState({}, '', url);
}

// 表单初始化
function initializeFormValues() {
    const params = getQueryParams();
    // Ensure elements exist before accessing their properties
    const searchInput = document.getElementById('searchInput');
    const statusFilter = document.getElementById('statusFilter');
    const limitFilter = document.getElementById('limitFilter');
    const sortFilter = document.getElementById('sortFilter');

    if (searchInput) searchInput.value = params.query;
    if (statusFilter) statusFilter.value = params.is_done === null ? '' : params.is_done.toString(); // Ensure value is string
    if (limitFilter) limitFilter.value = params.limit;
    if (sortFilter) sortFilter.value = params.sort;
}


// 时间处理函数
function formatDate(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    // Fallback to basic format if toLocaleString options fail in some environments
    try {
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    } catch (e) {
        console.warn("toLocaleString with options failed, using fallback format.");
        const year = date.getFullYear();
        const month = (date.getMonth() + 1).toString().padStart(2, '0');
        const day = date.getDate().toString().padStart(2, '0');
        const hours = date.getHours().toString().padStart(2, '0');
        const minutes = date.getMinutes().toString().padStart(2, '0');
        return `${year}-${month}-${day} ${hours}:${minutes}`;
    }
}


function calculateUrgency(task) {
    // Treat completed tasks as lowest urgency (highest number)
    if (task.completed) return Infinity;
    // Treat tasks without deadline as less urgent than those with past deadlines, but more than future ones
    if (!task.deadline) return Number.MAX_SAFE_INTEGER -1; // Assign a very large number, but less than Infinity

    const now = new Date();
    const deadline = new Date(task.deadline);
    const hoursLeft = (deadline - now) / (1000 * 60 * 60);
    return hoursLeft; // Negative values (past deadline) will naturally sort first
}


function formatTimeDiff(diffMs) {
    const totalSeconds = Math.floor(Math.abs(diffMs) / 1000);
    const days = Math.floor(totalSeconds / (3600 * 24));
    const hours = Math.floor((totalSeconds % (3600 * 24)) / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    let result = '';
    if (days > 0) result += `${days} 天 `;
    if (hours > 0) result += `${hours} 小时 `;
    if (minutes > 0 && days === 0) result += `${minutes} 分钟 `; // Show minutes only if less than a day
    if (seconds > 0 && days === 0 && hours === 0) result += `${seconds} 秒`; // Show seconds only if less than an hour

    return result.trim() || '0 秒'; // Handle case where diff is exactly 0
}


function getDeadlineStatus(deadline, completed) {
    if (completed) return { status: 'completed', class: 'text-gray-500' };
    if (!deadline) return { status: 'no-deadline', class: 'text-gray-400' }; // Style for no deadline

    const now = new Date();
    const deadlineDate = new Date(deadline);
    const hoursLeft = (deadlineDate - now) / (1000 * 60 * 60);

    if (hoursLeft < 0) return { status: 'overdue', class: 'text-red-600 font-bold' }; // Changed from 'error'
    if (hoursLeft <= 12) return { status: 'due-soon', class: 'text-yellow-600 font-bold' }; // Changed from 'warning'
    return { status: 'normal', class: 'text-gray-500' };
}


// 倒计时处理
function updateCountdown(element, deadline) {
    const now = new Date();
    const deadlineDate = new Date(deadline); // Ensure it's a Date object
    const diffMs = deadlineDate - now;

    let text = `截止: ${formatDate(deadline)}`;
    if (diffMs > 0) {
        text += ` (剩余 ${formatTimeDiff(diffMs)})`;
        element.textContent = text;
        // Return true to indicate countdown should continue
        return true;
    } else {
        text += ` (已超出 ${formatTimeDiff(Math.abs(diffMs))})`;
        element.textContent = text;
        // Return false to stop the interval and potentially trigger a refresh
        return false;
    }
}


// Task 列表处理
async function loadTasks(meetingId) {
    const params = getQueryParams();
    const taskListElement = document.getElementById('taskList');
    if (!taskListElement) {
        console.error("Element with ID 'taskList' not found.");
        return; // Stop execution if the main container is missing
    }
    
    if (!meetingId) {
        taskListElement.innerHTML = '<p class="text-center text-gray-500 my-4">请先选择一个会议</p>';
        return;
    }
    
    taskListElement.innerHTML = '<p class="text-gray-500 p-4">加载中...</p>'; // Show loading state

    try {
        // IMPORTANT: Adjust the fetch URL if your API endpoint is different
        const response = await fetch('/task/api', { // Make sure this endpoint is correct
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                action: 'list', // Ensure backend expects this action for listing
                list: params // Include meeting ID with filter params
            })
        });

        if (!response.ok) {
            // Handle HTTP errors (e.g., 404, 500)
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        console.log('Task data response:', data); // Add logging for debugging

        // More flexible handling of different response formats
        if (data.status === 'success') {
            let tasksToRender = [];
            
            // Handle various possible response formats
            if (Array.isArray(data.task_list)) {
                tasksToRender = data.task_list;
            } else if (data.tasks && Array.isArray(data.tasks)) {
                tasksToRender = data.tasks;
            } else if (data.data && Array.isArray(data.data)) {
                tasksToRender = data.data;
            } else if (typeof data.task_list === 'object' && data.task_list !== null) {
                // Handle nested object cases
                if (data.task_list.tasks && Array.isArray(data.task_list.tasks)) {
                    tasksToRender = data.task_list.tasks;
                } else if (data.task_list.data && Array.isArray(data.task_list.data)) {
                    tasksToRender = data.task_list.data;
                }
            } else if (Array.isArray(data)) {
                console.warn("Backend returned an array directly. Assuming it's the task list.");
                tasksToRender = data;
            }
            
            if (tasksToRender.length > 0) {
                renderTasks(tasksToRender);
            } else {
                taskListElement.innerHTML = '<p class="text-center text-gray-500 my-4">没有找到任务</p>';
            }
        } else {
            // Handle cases where the backend response format is unexpected
            console.error('Failed to load tasks: Invalid response format.', data);
            taskListElement.innerHTML = '<p class="text-red-500 p-4">加载任务失败：响应格式无效。</p>';
        }
    } catch (error) {
        console.error('Failed to load tasks:', error);
        taskListElement.innerHTML = `<p class="text-red-500 p-4">加载任务失败：${error.message}</p>`;
    }
}


function renderTasks(tasks) {
    const list = document.getElementById('taskList');
    const template = document.getElementById('taskTemplate');

    if (!list || !template) {
        console.error("Required elements 'taskList' or 'taskTemplate' not found.");
        return;
    }

    list.innerHTML = ''; // Clear previous content

    // Clear existing countdown timers
    if (window.countdownTimers) {
        window.countdownTimers.forEach(timer => clearInterval(timer));
    }
    window.countdownTimers = [];

    // Sort tasks based on the selected criteria
    const sortType = document.getElementById('sortFilter')?.value || 'urgency'; // Default to urgency
    tasks.sort((a, b) => {
        if (sortType === 'urgency') {
            return calculateUrgency(a) - calculateUrgency(b);
        } else if (sortType === 'deadline') {
            const deadlineA = a.deadline ? new Date(a.deadline).getTime() : Infinity;
            const deadlineB = b.deadline ? new Date(b.deadline).getTime() : Infinity;
            // Handle cases where one or both deadlines are null
            if (deadlineA === Infinity && deadlineB === Infinity) return 0;
            if (deadlineA === Infinity) return 1; // No deadline sorts last
            if (deadlineB === Infinity) return -1; // No deadline sorts last
            return deadlineA - deadlineB;
        } else { // created_at (assuming descending order - newest first)
            const createdA = a.created_at ? new Date(a.created_at).getTime() : 0;
            const createdB = b.created_at ? new Date(b.created_at).getTime() : 0;
            return createdB - createdA;
        }
    });


    if (tasks.length === 0) {
        list.innerHTML = '<p class="text-gray-500 p-4">没有找到符合条件的任务。</p>';
        return;
    }

    tasks.forEach(task => {
        // Ensure the template content exists before cloning
        if (!template.content) {
            console.error("Template element does not have content.");
            return;
        }
        const clone = template.content.cloneNode(true);
        const item = clone.querySelector('.task-item'); // Get the main item container

        if (!item) {
            console.error("Cloned template does not contain '.task-item'.");
            return;
        }

        // Safely query elements within the cloned item
        const checkbox = item.querySelector('.task-checkbox');
        const titleEl = item.querySelector('.task-title');
        const contentEl = item.querySelector('.task-content');
        const createdEl = item.querySelector('.task-created');
        const deadlineEl = item.querySelector('.task-deadline');

        item.dataset.id = task.id;
        // Store the original task data for editing
        item.dataset.task = JSON.stringify(task);

        if (checkbox) checkbox.checked = task.completed;
        if (titleEl) titleEl.textContent = task.title || '无标题'; // Provide default
        if (contentEl) contentEl.textContent = task.content || '';
        if (createdEl) createdEl.textContent = `创建: ${formatDate(task.created_at)}`;

        // Handle deadline display and countdown
        if (deadlineEl) {
            if (task.deadline) {
                const deadlineDate = new Date(task.deadline);
                const status = getDeadlineStatus(task.deadline, task.completed);
                deadlineEl.className = `task-deadline text-xs ${status.class}`; // Reset classes

                const now = new Date();
                const hoursLeft = (deadlineDate - now) / (1000 * 60 * 60);
                // Start countdown only for non-completed tasks due within the next 2 hours
                const needsCountdown = !task.completed && hoursLeft > 0 && hoursLeft <= 2;

                if (needsCountdown) {
                    if (updateCountdown(deadlineEl, deadlineDate)) { // Initial update
                        const timer = setInterval(() => {
                            if (!updateCountdown(deadlineEl, deadlineDate)) {
                                clearInterval(timer); // Stop timer if deadline passed
                                // Optionally refresh task list slightly after deadline passes
                                setTimeout(loadTasks, 1500);
                            }
                        }, 60000); // Update countdown every minute
                        window.countdownTimers.push(timer);
                    }
                } else {
                    // Static display for completed, overdue, far future, or no deadline tasks
                    let text = `截止: ${formatDate(task.deadline)}`;
                     if (!task.completed) {
                         const diffMs = deadlineDate - now;
                         if (diffMs < 0) {
                             text += ` (已超出 ${formatTimeDiff(Math.abs(diffMs))})`;
                         } else {
                             // Optionally show remaining time even if > 2 hours
                             // text += ` (剩余 ${formatTimeDiff(diffMs)})`;
                         }
                     }
                    deadlineEl.textContent = text;
                }
            } else {
                deadlineEl.textContent = '无截止日期';
                deadlineEl.className = 'task-deadline text-xs text-gray-400';
            }
        }


        // Apply styling for completed tasks
        if (task.completed) {
            if (titleEl) titleEl.classList.add('line-through', 'text-gray-500');
            if (contentEl) contentEl.classList.add('line-through', 'text-gray-500');
            if (item) item.classList.add('opacity-70'); // Dim completed items slightly
        } else {
             if (titleEl) titleEl.classList.remove('line-through', 'text-gray-500');
             if (contentEl) contentEl.classList.remove('line-through', 'text-gray-500');
             if (item) item.classList.remove('opacity-70');
        }

        list.appendChild(clone);
    });
}


// 对话框处理
function openAddDialog() {
    const dialog = document.getElementById('addDialog');
    const form = document.getElementById('addForm');
    if (dialog && form) {
        form.reset(); // Clear previous input
        dialog.classList.remove('hidden');
        dialog.classList.add('flex'); // Use flex to center
        // Optional: Focus the first input field
        form.querySelector('input[name="title"]')?.focus();
    } else {
        console.error("Add dialog or form not found");
    }
}

function closeAddDialog() {
    const dialog = document.getElementById('addDialog');
    if (dialog) {
        dialog.classList.add('hidden');
        dialog.classList.remove('flex');
    }
}

function openEditDialog(task) {
    const dialog = document.getElementById('editDialog');
    const form = document.getElementById('editForm');

    if (dialog && form && task) {
        // Populate form fields safely
        form.id.value = task.id || '';
        form.title.value = task.title || '';
        form.content.value = task.content || '';
        // Format deadline for datetime-local input (YYYY-MM-DDTHH:mm)
        form.deadline.value = task.deadline ? task.deadline.slice(0, 16) : '';

        dialog.classList.remove('hidden');
        dialog.classList.add('flex');
        form.querySelector('input[name="title"]')?.focus();
    } else {
         console.error("Edit dialog, form, or task data missing");
    }
}


function closeEditDialog() {
    const dialog = document.getElementById('editDialog');
    if (dialog) {
        dialog.classList.add('hidden');
        dialog.classList.remove('flex');
    }
}

// 事件处理
function debounce(func, delay) {
    let timer;
    return function(...args) {
        clearTimeout(timer);
        timer = setTimeout(() => func.apply(this, args), delay);
    }
}


// Debounced handler for filter changes
const handleFilterChange = debounce(() => {
    const query = document.getElementById('searchInput')?.value || '';
    const statusValue = document.getElementById('statusFilter')?.value;
    const limit = parseInt(document.getElementById('limitFilter')?.value) || 10;
    const sort = document.getElementById('sortFilter')?.value || 'urgency';

    // Convert status filter value to boolean or null
    let is_done = null;
    if (statusValue === 'true') is_done = true;
    if (statusValue === 'false') is_done = false;

    const params = { query, is_done, limit, sort };
    updateQueryParams(params); // Update URL
    loadTasks(); // Reload tasks with new filters
}, 300); // 300ms delay


// 在文件顶部添加定时器变量
let autoRefreshTimer;

// 初始化事件监听
document.addEventListener('DOMContentLoaded', () => {
    // --- Dialog Buttons ---
    document.getElementById('addTaskBtn')?.addEventListener('click', openAddDialog);
    document.getElementById('addDialogCancel')?.addEventListener('click', closeAddDialog);
    document.getElementById('editDialogCancel')?.addEventListener('click', closeEditDialog);
    // Close dialog if clicking outside the content area
    document.getElementById('addDialog')?.addEventListener('click', (e) => {
        if (e.target === e.currentTarget) closeAddDialog();
    });
     document.getElementById('editDialog')?.addEventListener('click', (e) => {
        if (e.target === e.currentTarget) closeEditDialog();
    });


    // --- Add Task Form ---
    const addForm = document.getElementById('addForm');
    if (addForm) {
        addForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const task = {
                title: form.title.value.trim(),
                content: form.content.value.trim(),
                // Get deadline value, ensure it's null if empty
                deadline: form.deadline.value || null
            };

            // Basic validation
            if (!task.title) {
                alert('任务标题不能为空！');
                return;
            }

            try {
                const response = await fetch('/task/api', { // Ensure endpoint is correct
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        action: 'add', // Ensure backend expects this action
                        task: task
                    })
                });
                 if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
                const data = await response.json();
                if (data.status === 'success') {
                    closeAddDialog();
                    loadTasks(); // Refresh list
                } else {
                     console.error('Failed to add task:', data.error || 'Unknown error');
                     alert(`添加任务失败: ${data.error || '未知错误'}`);
                }
            } catch (error) {
                console.error('Failed to add task:', error);
                alert(`添加任务时出错: ${error.message}`);
            }
        });
    } else {
        console.warn("Add form not found.");
    }

    // --- Task List Event Delegation ---
    const taskList = document.getElementById('taskList');
    if (taskList) {
        taskList.addEventListener('change', async (e) => {
            // Handle Checkbox Change (Update Status)
            if (e.target.matches('.task-checkbox')) {
                const item = e.target.closest('.task-item');
                if (!item) return;
                const id = item.dataset.id;
                const completed = e.target.checked;

                try {
                    const response = await fetch('/task/api', { // Ensure endpoint is correct
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            action: 'update', // Ensure backend expects this action
                            task: { id, completed } // Send only changed fields
                        })
                    });
                     if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
                    const data = await response.json();
                    if (data.status === 'success') {
                        loadTasks(); // Refresh list to show visual changes
                    } else {
                        console.error('Failed to update task status:', data.error || 'Unknown error');
                        alert(`更新任务状态失败: ${data.error || '未知错误'}`);
                        e.target.checked = !completed; // Revert checkbox on failure
                    }
                } catch (error) {
                    console.error('Failed to update task status:', error);
                     alert(`更新任务状态时出错: ${error.message}`);
                    e.target.checked = !completed; // Revert checkbox on failure
                }
            }
        });

        taskList.addEventListener('click', async (e) => {
            // Handle Delete Button Click
            const deleteButton = e.target.closest('.delete-btn');
            if (deleteButton) {
                const item = deleteButton.closest('.task-item');
                 if (!item) return;
                const id = item.dataset.id;
                const title = item.querySelector('.task-title')?.textContent || '此任务';

                if (!confirm(`确定要删除任务 "${title}" 吗？`)) return;

                try {
                    const response = await fetch('/task/api', { // Ensure endpoint is correct
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            action: 'delete', // Ensure backend expects this action
                            task: { id }
                        })
                    });
                     if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
                    const data = await response.json();
                    if (data.status === 'success') {
                        loadTasks(); // Refresh list
                    } else {
                         console.error('Failed to delete task:', data.error || 'Unknown error');
                         alert(`删除任务失败: ${data.error || '未知错误'}`);
                    }
                } catch (error) {
                    console.error('Failed to delete task:', error);
                     alert(`删除任务时出错: ${error.message}`);
                }
            }

            // Handle Edit Button Click
            const editButton = e.target.closest('.edit-btn');
            if (editButton) {
                const item = editButton.closest('.task-item');
                 if (!item || !item.dataset.task) return;
                try {
                    const task = JSON.parse(item.dataset.task);
                    openEditDialog(task);
                } catch (parseError) {
                    console.error("Failed to parse task data for editing:", parseError);
                    alert("无法加载任务数据进行编辑。");
                }
            }
        });
    } else {
        console.warn("Task list container not found.");
    }


    // --- Edit Task Form ---
    const editForm = document.getElementById('editForm');
    if (editForm) {
        editForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const task = {
                id: form.id.value,
                title: form.title.value.trim(),
                content: form.content.value.trim(),
                deadline: form.deadline.value || null // Ensure null if empty
            };

             if (!task.title) {
                alert('任务标题不能为空！');
                return;
            }

            try {
                const response = await fetch('/task/api', { // Ensure endpoint is correct
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        action: 'update', // Ensure backend expects this action
                        task: task // Send all fields for update
                    })
                });
                 if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
                const data = await response.json();
                if (data.status === 'success') {
                    closeEditDialog();
                    loadTasks(); // Refresh list
                } else {
                     console.error('Failed to update task:', data.error || 'Unknown error');
                     alert(`更新任务失败: ${data.error || '未知错误'}`);
                }
            } catch (error) {
                console.error('Failed to update task:', error);
                 alert(`更新任务时出错: ${error.message}`);
            }
        });
    } else {
        console.warn("Edit form not found.");
    }

    // --- Filter Event Listeners ---
    document.getElementById('searchInput')?.addEventListener('input', handleFilterChange);
    document.getElementById('statusFilter')?.addEventListener('change', handleFilterChange);
    document.getElementById('limitFilter')?.addEventListener('change', handleFilterChange);
    document.getElementById('sortFilter')?.addEventListener('change', handleFilterChange);

    // --- Browser History Navigation ---
    window.addEventListener('popstate', () => {
        initializeFormValues(); // Update form controls based on URL
        loadTasks(); // Load tasks based on URL state
    });

    // --- Auto Refresh ---
    // Clear previous timer if script re-runs (e.g., during development)
    if (window.autoRefreshTimerId) {
        clearInterval(window.autoRefreshTimerId);
    }
    // Set a new timer and store its ID globally
    window.autoRefreshTimerId = setInterval(loadTasks, 30000); // Refresh every 30 seconds


    // --- Cleanup Timers on Page Unload ---
    window.addEventListener('beforeunload', () => {
        if (window.countdownTimers) {
            window.countdownTimers.forEach(timer => clearInterval(timer));
        }
        if (window.autoRefreshTimerId) {
            clearInterval(window.autoRefreshTimerId);
        }
    });

    // --- Initial Load ---
    loadMeetings();
    loadURLState(); // loadURLState 会处理初始 Tab 和 Meeting 的加载
    initializeFormValues(); // Set initial form values from URL or defaults
    loadTasks(); // Initial task load
});

async function generateTasksFromSummary(meetingId) {
  try {
      const response = await fetch('/task/api', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
              action: "generate_from_summary",
              list: { meeting_id: meetingId }
          })
      });

      // 检查网络响应状态
      if (!response.ok) {
          const errorText = await response.text();
          throw new Error(`HTTP ${response.status}: ${errorText}`);
      }

      const responseData = await response.json();
      
      // 检查业务状态
      if (responseData.status !== 'success') {
          throw new Error(responseData.error || '未知错误');
      }

      // 解析任务数据（适配两种格式）
      const rawTasks = responseData.task_list?.Choices?.[0]?.Message?.Content?.StringValue || 
                      responseData.task_list || 
                      [];
      
      // 格式化任务数据
      const formattedTasks = typeof rawTasks === 'string' ? 
          parseTasksFromString(rawTasks) : 
          rawTasks.map(t => ({
              title: t.title || '新任务',
              content: t.description || t.content || '',
              deadline: t.due_date || null
          }));

      // 渲染任务列表
      renderTasks(formattedTasks);
      
      // 自动切换到任务标签页
      switchTab('task');
      
      // 显示成功提示
      showToast('任务生成成功', 'success');
      
  } catch (error) {
      console.error('生成任务失败:', error);
      showToast(`生成失败: ${error.message}`, 'error');
  }
}

// 辅助函数：解析字符串格式的任务
function parseTasksFromString(taskString) {
  const taskRegex = /(\d+)\.\s*任务描述：([^，]+)，负责人：([^，]+)，完成时间：([^\n]+)/g;
  const matches = [...taskString.matchAll(taskRegex)];
  
  return matches.map(match => ({
      title: match[2].trim(),
      content: `负责人：${match[3].trim()}`, 
      deadline: match[4].trim() !== '未提及' ? match[4].trim() : null
  }));
}

// 辅助函数：显示通知
function showToast(message, type = 'info') {
  const toast = document.createElement('div');
  toast.className = `fixed bottom-4 right-4 px-6 py-3 rounded-lg shadow-lg 
                    ${type === 'success' ? 'bg-green-500 text-white' : 
                     type === 'error' ? 'bg-red-500 text-white' : 
                     'bg-blue-500 text-white'}`;
  toast.textContent = message;
  
  document.body.appendChild(toast);
  setTimeout(() => toast.remove(), 3000);
}