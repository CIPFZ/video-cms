import React, { useEffect, useState } from 'react';
import { Alert, List, Segmented, Space, Tag, Typography } from 'antd';
import { t, useI18n } from '@/i18n';
import { listCollectionRunItems, type CollectionRun, type CollectionRunItem } from '@/services/media';
import { playerURL } from '../shared';

const actionColors: Record<string, string> = { inserted: 'green', updated: 'blue', failed: 'red' };
const actionLabel = (action: string) => action === 'inserted' ? t('cms.media.runItems.action.inserted') : action === 'updated' ? t('cms.media.runItems.action.updated') : t('cms.media.runItems.action.failed');

// Titles a run inserted, updated or failed to save; reloads as a running task's counters move.
export default function RunItems({ run }: { run: CollectionRun }) {
 useI18n();
 const [action,setAction]=useState(''),[page,setPage]=useState(1),[rows,setRows]=useState<CollectionRunItem[]>([]),[total,setTotal]=useState(0),[summary,setSummary]=useState<Record<string,number>>({}),[loading,setLoading]=useState(false),[failed,setFailed]=useState(false);
 const version=run.insertedCount+run.updatedCount+run.errorCount;
 useEffect(()=>{setAction('');setPage(1)},[run.id]);
 useEffect(()=>{let live=true;setLoading(true);listCollectionRunItems(run.id,{page,pageSize:20,action:action||undefined}).then(r=>{if(!live)return;if(r.code!==0)throw Error();setRows(r.data?.list||[]);setTotal(r.data?.total||0);setSummary(r.data?.summary||{});setFailed(false)}).catch(()=>live&&setFailed(true)).finally(()=>live&&setLoading(false));return()=>{live=false}},[run.id,action,page,version]);
 const count=(key:string)=>summary[key]||0, all=count('inserted')+count('updated')+count('failed');
 // Runs from before item logging keep only their counters.
 if(!loading&&!failed&&all===0&&version>0) return <Alert showIcon type="info" message={t('cms.media.runItems.legacy')}/>;
 return <Space direction="vertical" size={12} style={{width:'100%'}}>
 {failed&&<Alert showIcon type="warning" message={t('cms.media.runItems.loadFailed')}/>}
 <Segmented value={action} onChange={v=>{setAction(String(v));setPage(1)}} options={[
  {value:'',label:t('cms.media.runItems.all',{count:all})},{value:'inserted',label:t('cms.media.runItems.inserted',{count:count('inserted')})},
  {value:'updated',label:t('cms.media.runItems.updated',{count:count('updated')})},{value:'failed',label:t('cms.media.runItems.failed',{count:count('failed')})},
 ]}/>
 <List<CollectionRunItem> size="small" bordered loading={loading} dataSource={rows} locale={{emptyText:t('cms.media.runItems.empty')}}
  pagination={total>20?{current:page,pageSize:20,total,size:'small',showSizeChanger:false,onChange:setPage}:false}
  renderItem={item=><List.Item><Space direction="vertical" size={2} style={{width:'100%'}}>
   <Space><Tag color={actionColors[item.action]}>{actionLabel(item.action)}</Tag>{item.itemId?<Typography.Link href={playerURL(item.itemId)} target="_blank">{item.title||item.externalId}</Typography.Link>:<Typography.Text>{item.title||item.externalId}</Typography.Text>}</Space>
   <Typography.Text type="secondary" style={{fontSize:12}}>{t('cms.media.runItems.externalId',{id:item.externalId})}</Typography.Text>
   {item.errorMessage&&<Typography.Text type="danger">{item.errorMessage}</Typography.Text>}
  </Space></List.Item>}/>
 </Space>;
}
