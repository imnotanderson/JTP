using UnityEngine;
using System.Collections;
using System.Net.Sockets;
using System.Net;
using System.Threading;
using System;
using System.Collections.Generic;


public class Receiver  {

    IPEndPoint raddr;
    Socket socket;
    Session session;
    Thread thRecv = null;
    int timeout = 5000;

    public int DataCount
    {
        get
        {
            if (session == null)
            {
                return 0;
            }
            return session.recvBuff.Count;
        }
    }

    public void Dial(string addr,int port)
    {
        raddr = new IPEndPoint(IPAddress.Parse(addr), port);
        socket = new Socket(AddressFamily.InterNetwork, SocketType.Dgram, ProtocolType.Udp);
        session = new Session(raddr);
        thRecv = new Thread(Recv);
        thRecv.Start();
    }

    public void Close() {
        try
        {
            thRecv.Abort();
        }
        catch(Exception e) {
            Log.Info("{0}", e);
        }
        finally
        {
            Log.Info("receiver close");
        }
    }
    
    void Recv()
    {
        try
        {
            socket.SetSocketOption(SocketOptionLevel.Socket, SocketOptionName.ReceiveTimeout, timeout);
            var firstData = getReplyData(0, 0);
            socket.SendTo(firstData, raddr);
            for (;;)
            {
                var data = new byte[1024];
                EndPoint addr = this.raddr;
                var n = socket.ReceiveFrom(data, ref addr);
                var id = bytes2uint(data);
                var newData = new byte[n - 4];
                for (int i = 0; i < newData.Length; i++)
                {
                    newData[i] = data[i + 4];
                }
                var segment = new Segment(id, newData, this.raddr);
                var pSession = getSession(segment);
                if (pSession == null)
                {
                    continue;
                }
                if (pSession.checkSegment(segment) == false)
                {
                    continue;
                }
                appendSegment(segment);
                reply(segment);
            }
        }catch(Exception e)
        {
            Log.Info("Recv err:{0}", e);
        }
    }
    
    byte[] getReplyData(uint nextId,uint maxId)
    {
        byte[] b = new byte[8];
        uint2bytes(nextId).CopyTo(b, 0);
        uint2bytes(maxId).CopyTo(b, 4);
        return b;
    }

    uint bytes2uint(byte[] b)
    {
        if (BitConverter.IsLittleEndian == false)
        {
            Array.Reverse(b);
        }
        return BitConverter.ToUInt32(b, 0);
    }

    byte[] uint2bytes(uint i)
    {

        var b = System.BitConverter.GetBytes(i);
        if (BitConverter.IsLittleEndian == false)
        {
            Array.Reverse(b);
        }
        return b;
    }

    Session getSession(Segment segment)
    {
        if (this.session == null)
        {
            this.session = new Session(segment.addr);
            return session;
        }
        if (this.session.checkAddr(segment) == false)
        {
            return null;
        }
        return session;
    }


    void appendSegment(Segment segment)
    {
        session.appendAndSort(segment);
    }

    void reply(Segment segment) {
        if (session == null)
        {
            Log.Info("reply session nil");
            return;
        }
        var data = getReplyData(session.nextId, session.maxId);
        Log.Info("s.sendReply nextId:{0} maxId:{1}", session.nextId, session.maxId);
        if (session.raddr.ToString() != this.raddr.ToString())
        {
            Log.Info("pSession.raddr.String()!=r.pConn.RemoteAddr() return");
            return;
        }
        try
        {
            socket.SendTo(data, raddr);
        }
        catch(Exception e)
        {
            Log.Info("sendTo err:{0}", e);
        }
    }

    public byte[] Read() {
        if (this.session == null)
        {
            return null;
        }
        if (this.session.recvBuff.Count > 0)
        {
            return this.session.Read();
        }
        return null;
    }
}


public class Session
{
    public IPEndPoint raddr;
    public List<Segment> list;
    public uint nextId;
    public uint maxId;
    public List<byte> recvBuff;
    object mLock;

    public Session(IPEndPoint raddr)
    {
        this.raddr = raddr;
        list = new List<Segment>();
        recvBuff = new List<byte>();
        mLock = new object();
    }

    public bool checkSegment(Segment segment)
    {
        if (segment.id < this.nextId)
        {
            return false;
        }
        foreach (var v in list)
        {
            if (v.id == segment.id)
            {
                return false;
            }
        }
        return true;
    }

    public void appendAndSort(Segment segment)
    {
        var insertIdx = list.Count;
        for (int idx = 0; idx < list.Count; idx++)
        {
            var v = list[idx];
            if (v.id > segment.id)
            {
                insertIdx = idx;
                break;
            }
        }
        Log.Info("insertIdx:{0}", insertIdx);
        Log.Info("before:{0}", list);
        if (insertIdx == list.Count)
        {
            list.Add(segment);
        }
        else
        {
            //check
            list.Insert(insertIdx, segment);
        }
        Log.Info("after:{0}", list);
        if (maxId < segment.id)
        {
            maxId = segment.id;
        }
        for (int i = 0; i < list.Count; i++)
        {
            if (nextId == list[i].id)
            {
                lock (mLock)
                {
                    recvBuff.AddRange(list[i].data);
                }
                list.RemoveAt(0);
                nextId++;
                i--;
            }
            else
            {
                break;
            }
        }
    }

    public byte[] Read()
    {
        byte[] data = null;
        lock (mLock)
        {
            data = recvBuff.ToArray();
            recvBuff.Clear();
        }
        return data;
    }

    public bool checkAddr(Segment other)
    {
        if (other == null) return false;
        if (this.raddr.Port == other.addr.Port && this.raddr.Address.Equals(other.addr.Address))
        {
            return true;
        }
        return false;
    }

}

public class Segment
{
    public uint id;
    public byte[] data;
    public IPEndPoint addr;

    public Segment(uint id,byte[] data,IPEndPoint addr) {
        this.id = id;
        this.data = data;
        this.addr = addr;
    }

    public override string ToString()
    {
        return addr.ToString();
    }
}

public class Log
{
    public static void Info(string format,params object[] args)
    {
        return;
        Debug.Log(string.Format(format, args));
        UnityEngine.MonoBehaviour.print(string.Format(format, args));
    }
}