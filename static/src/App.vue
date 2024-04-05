<!--  三段式：头部、主体、底部 -->
<template>
  <el-container style="margin: 0">

    <el-header style="margin-bottom: 4em;margin-top: 0 ">
      <div class="i18n_btn">
        <el-button @click="toggleLanguage">{{ state.isEnglish ? '中文' : 'English' }}</el-button>
      </div>

      <p class="text-center title">{{ state.isEnglish ? i18n.title_first.en : i18n.title_first.zh }}
        <br>{{ state.isEnglish ? i18n.title_second.en : i18n.title_second.zh }}
      </p>
    </el-header>

    <el-main style="margin-top: 1em">
      <!-- github 开源标识 -->
      <div class="github-badge-container">
        <a href="https://github.com/Bistutu/GoMusic" target="_blank"><img
            src="https://img.shields.io/github/stars/Bistutu/GoMusic?style=flat-square&logo=github&label=Star"
            alt="GitHub stars"></a>
      </div>

      <el-row justify="center" @submit.prevent="fetchLinkDetails">
        <el-col :md="12">
          <el-form-item >
            <el-input v-model="state.link" size="large"
                      :placeholder="state.isEnglish ? i18n.inputPlaceholder.en : i18n.inputPlaceholder.zh"
                      @keyup.enter="fetchLinkDetails">
            </el-input>
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item>
        <el-button type="danger" class="button-center lang-song-list-btn" @click="fetchLinkDetails">
          {{ state.isEnglish ? i18n.fetchSongList.en : i18n.fetchSongList.zh }}
        </el-button>
      </el-form-item>

      <el-row justify="center">
        <el-col :md="12">
          <el-form-item>
            <el-input type="textarea" v-model="state.result" :rows="15"
                      :placeholder="state.isEnglish ? i18n.resultHint.en : i18n.resultHint.zh"></el-input>
          </el-form-item>

          <el-form-item>
            <div class="songs-count-display text-center" v-show="state.songsCount > 0">
              {{ state.isEnglish ? i18n.songsCount.en : i18n.songsCount.zh }}: {{ state.songsCount }}
            </div>
          </el-form-item>

        </el-col>
      </el-row>

      <el-form-item>
        <el-button @click="copyResult" class="button-center lang-copy-btn">
          {{ state.isEnglish ? i18n.copy.en : i18n.copy.zh }}
        </el-button>
      </el-form-item>

    </el-main>

    <el-footer>
      <el-row justify="center">
        <el-col :md="12">
          <el-collapse v-model="activeNames">
            <el-collapse-item :title="state.isEnglish ? i18n.guide.en : i18n.guide.zh" name="first">
              <ol>
                <li>{{ state.isEnglish ? i18n.guideFirst.en : i18n.guideFirst.zh }}</li>
                <li>{{ state.isEnglish ? i18n.guideSecond.en : i18n.guideSecond.zh }}</li>
                <li>{{ state.isEnglish ? i18n.guideThird_1.en : i18n.guideThird_1.zh }}
                  <b><a :href="state.isEnglish ? i18n.TunemyMusicUrl.en: i18n.TunemyMusicUrl.zh" target="_blank">TunemyMusic</a></b>
                  {{ state.isEnglish ? i18n.guideThird_2.en : i18n.guideThird_2.zh }}
                </li>
                <li>{{ state.isEnglish ? i18n.guideFourth.en : i18n.guideFourth.zh }}</li>
              </ol>
            </el-collapse-item>
          </el-collapse>
        </el-col>
      </el-row>
    </el-footer>
  </el-container>

</template>

<script setup>
import {reactive, ref} from 'vue';
import axios from 'axios';
import {ElMessage} from 'element-plus';
import {isSupportedPlatform, isValidUrl} from "@/utils/utils";
import {sendErrorMessage, sendSuccessMessage} from "@/utils/tip";

const activeNames = ref(['first']);
const state = reactive({
  link: '',
  result: '',
  isEnglish: false,
  songsCount: 0,
});

const i18n = {
  title_first: {
    en: 'Migrate Netease/QQ Music Playlist',
    zh: '迁移网易云/QQ音乐歌单',
  },
  title_second: {
    en: 'To Apple/Youtube/Spotify Music',
    zh: '至 Apple/Youtube/Spotify Music',
  },
  inputPlaceholder: {
    en: 'Enter any playlist link, such as: http://163cn.tv/zoIxm3',
    zh: '输入任意歌单链接，如：http://163cn.tv/zoIxm3',
  },
  fetchSongList: {
    en: 'Fetch Song List',
    zh: '获取歌单',
  },
  resultHint: {
    en: 'Result will be displayed here',
    zh: '结果会显示在这里',
  },
  songsCount: {
    en: 'Songs Count',
    zh: '歌曲总数',
  },
  guide: {
    en: 'Guide',
    zh: '《使用指南》',
  },
  guideFirst: {
    en: 'Enter playlist link, such as: http://163cn.tv/zoIxm3',
    zh: '输入歌单链接，如：http://163cn.tv/zoIxm3',
  },
  guideSecond: {
    en: 'Copy the query result',
    zh: '复制查询结果',
  },
  guideThird_1: {
    en: 'Open ',
    zh: '打开 ',
  },
  guideThird_2: {
    en: ' website',
    zh: ' 网站',
  },
  TunemyMusicUrl: {
    en: 'https://www.tunemymusic.com',
    zh: 'https://www.tunemymusic.com/zh-CN/transfer',
  },
  guideFourth: {
    en: 'Select playlist source as "Any Text", paste the copied playlist, select Apple/Youtube/Spotify Music as destination, confirm migration',
    zh: '选择歌单来源“任意文本”，将刚刚复制的歌单粘贴进去，选择 Apple/Youtube/Spotify Music 作为目的地，确认迁移',
  },
  about: {
    en: 'About author',
    zh: '关于作者',
  },
  copy: {
    en: 'Copy result',
    zh: '复制结果',
  },
  noContent: {
    en: 'No content to copy',
    zh: '没有内容可复制',
  },
  copied: {
    en: 'Copied to clipboard',
    zh: '已复制到剪贴板',
  },
}


function reset(msg) {
  sendErrorMessage(msg)
  state.result = ""
  state.songsCount = 0
}

// 获取歌单详情
const fetchLinkDetails = async () => {

  state.link = state.link.trim();

  if (!isValidUrl(state.link) || !isSupportedPlatform(state.link)) {
    reset(state.isEnglish ? 'Invalid link, only support Netease and QQ Music' : '链接无效，平台仅支持网易云音乐和QQ音乐');
    return;
  }

  const params = new URLSearchParams();
  params.append('url', state.link);

  try {
    // const resp = await axios.post('https://music.unmeta.cn/songlist', params, {
    const resp = await axios.post('https://sss.unmeta.cn/songlist', params, {
      headers: {'Content-Type': 'application/x-www-form-urlencoded'}
    });

    console.log(resp.data)
    if (resp.data.code !== 1) {
      reset(state.isEnglish ? "Request failed, please try again later~" : "请求失败，请稍后再试~");
      return;
    }
    sendSuccessMessage(state.isEnglish ? "Song list fetched successfully" : "歌单获取成功");
    state.result = resp.data.data.songs.join('\n')
    state.songsCount = resp.data.data.songs_count;
  } catch (err) {
    console.error(err);
    // 后端规定的错误格式 err.response.data.msg
    reset(err.response.data.msg || state.isEnglish ? "Request failed, please try again later~" : "请求失败，请稍后再试~");
  }
};

// 复制结果
const copyResult = () => {
  if (!state.result) {
    ElMessage.error({message: state.isEnglish ? 'No content to copy' : '没有内容可复制', type: 'error'});
    return;
  }
  const textarea = document.createElement('textarea');
  textarea.value = state.result;
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand('copy');
  document.body.removeChild(textarea);
  ElMessage.success({message: state.isEnglish ? 'Copied to clipboard' : '已复制到剪贴板', type: 'success'});
};

const toggleLanguage = () => {
  state.isEnglish = !state.isEnglish;
};

</script>


<style>
.text-center {
  text-align: center;
}

.songs-count-display {
  margin-top: -1.25em;
  color: #333;
  height: 1em;
  width: 100%;
}

.github-badge-container {
  text-align: center; /* 水平居中 */
  margin-bottom: 1em; /* 添加一些上下边距 */
}

.i18n_btn {
  text-align: right;
  margin-right: 0.5em !important;
  margin-bottom: 0;
}

.title {
  font-size: 2em;
  margin-top: 0 !important;
}

@media (max-width: 768px) {
  .title {
    font-size: 1.5em !important;
  }
}


.button-center {
  margin: 0 auto;
  display: block; /* 按钮水平居中 */
}

.el-collapse-item__header {
  font-size: 1em !important;
}


.el-input {
  margin: 0 auto;
  display: flex !important; /* 输入框水平居中 */
}

.el-collapse-item ol li {
  font-size: medium;
}

</style>