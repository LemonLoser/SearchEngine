package bplustree

import (
  "unsafe"
)

const (
  LEAFPAGE = 0
  INTERPAGE = 1
)
/*
** An instance of this object represents a single database file.
**
** A single database file can be in use at the same time by two
** or more database connections.  When two or more connections are
** sharing the same database file, each connection has it own
** private Btree object for the file and each of those Btrees points
** to this one BPlusTree object.
*/
type BPlusTree struct {
  Pager *pPager           /* The page cache */
  MemPage *page           /* First page of the database */
  uint16 maxLocal         /* Maximum local payload in non-LEAFDATA tables */
  uint16 minLocal         /* Minimum local payload in non-LEAFDATA tables */
  uint16 maxLeaf          /* Maximum local payload in a LEAFDATA table */
  uint16 minLeaf          /* Minimum local payload in a LEAFDATA table */
  uint32 pageSize         /* Total number of bytes on a page */
  uint32 usableSize       /* Number of usable bytes on each page */
  uint32 nPage            /* Number of pages in the database */
  hm map[uint32]*MemPage  /* map pageno to MemPage */
}

/* Each btree pages is divided into three sections:  The header, the
** cell pointer array, and the cell content area.  Page 1 also has a 100-byte
** file header that occurs before the page header.
**
**      |----------------|
**      | file header    |   100 bytes.  Page 1 only.
**      |----------------|
**      | page header    |   8 bytes for leaves.  12 bytes for interior nodes
**      |----------------|
**      | cell pointer   |   |  2 bytes per cell.  Sorted order.
**      | array          |   |  Grows downward
**      |                |   v
**      |----------------|
**      | unallocated    |
**      | space          |
**      |----------------|   ^  Grows upwards
**      | cell content   |   |  Arbitrary order interspersed with freeblocks.
**      | area           |   |  and free space fragments.
**      |----------------|
**
** The page headers looks like this:
**
**   OFFSET   SIZE     DESCRIPTION
**      0       1      Flags. 1: interpage, 2: leafpage, 4: overflowpage, 8: zeropage
**      1       2      byte offset to the first freeblock
**      3       2      number of cells on this page
**      5       2      first byte of the cell content area
**      7       1      number of fragmented free bytes
**      8       4      Right child (the Ptr(N) value).  Omitted on leaves.
*/
type PageHeader struct {
  flag uint8
  freeOffset uint16
  nCell uint16
  pgno uint16
  nFree uint8
  parent uint32
}

type MemPage struct{
  ph *PageHeader
  aData unsafe.Pointer          /* Pointer to disk image of the page data */
  aDataEnd unsafe.Pointer       /* One byte past the end of usable data */
  cell unsafe.Pointer       /* The cell index area */
  aDataOfst unsafe.Pointer      /* Same as aData for leaves.  aData+4 for interior */
}


/* The basic idea is that each page of the file contains N database
** entries and N+1 pointers to subpages.
**
**   --------------------------------------------------------------
**   |  Ptr(0) | Key(0) | Ptr(1) | Key(1) | ... | Key(N) | Ptr(N) |
**   --------------------------------------------------------------
*/
type Cell struct {
  ptr      uint32      /* page number or Offset of the page start of a payload */
  key      uint32      /* The key for Payload*/
}

/* DocId1 DocId2 ...
**   -----------------------------------------------------------------
**   |  key | DocId1 | DocId1 | DocId3 | ... | DocId(N-1) | DocId(N) |
**   -----------------------------------------------------------------
 */
type Payload struct {
  key     uint32             /* value in the unpacked key */
  size    uint16             /* Number of values.  Might be zero */
  entrys  unsafe.Pointer            /* fot data compress */
}

func (bptree *BPlusTree) Insert(pl *PlayLoad) {
  offset, pg := Search(pl.key)
  if offset != nil {
    return
  }

  ok, key, newpg := insert(pl, pg)
  if ok != nil {
    return
  }

  ppg := bptree.hm[pg.parent()]

  for {
    ok, key, newpg = insert(&Cell{key: key,ptr: newpg.ph.phno}, ppg)
    if ok != nil {
      return
    }

    if ppg.ph.pgno == bptree.page.pgno {
      // alloc new root page for bplustree and update bplustree page
      rootpage := &MemPage{}
      bptree.page = rootpage
      // insert new page cell
      _, _, _ := insert(&Cell{key: key,ptr: newpg.ph.phno}, rootpage)

      // insert origin page cell
      _, _, _ := insert(&Cell{key: key,ptr: ppg.ph.phno}, rootpage)
      return
    }
    ppg = bptree.hm[ppg.parent()]
  }
}

func (bptree *BPlusTree) Search(key int) (uint16, *MemPage) {
  curr := bptree.pPage
  for {
    switch t := curr.ph.flag {
    case LEAFPAGE:
      offset, ok := find(curr, key)
      if !ok {
        return nil, curr
      }
      return offset, curr
    case INTERPAGE:
      pgno, _ := find(key)
      curr = bptree.hm[pgno]
      // pager should load page and cached
    default:
      panic("no such flag!")
    }
  }
}

func (p *MemPage) insert(data interface{}) (bool, uint32, *MemPage){
  ok := p.full(data)
  if !ok {
    return true, nil, nil
  }

  //key, newpg :=split(pg)
  newpg := newpage()
  //update page info

  return false, key, newpg
}

func (p *MemPage) find(key int) (int, bool) {
  cmp := func (i int) bool {
    return p.cell[i].key >= key
  }

  i := sort.Search(p.ph.nCell, cmp)

  if p.ph.flag == INTERPAGE {
    return p.cell[i].ptr, true
  }

  if i <= p.ph.nCell && p.cell[i].key == key {
    return p.cell[i].ptr, true
  }

  return nil, false
}

func (p *MemPage) parent() uint32 {
  return p.ph.pgno
}

func (p *MemPage) setparent(uint32 pgno) {
  p.ph.parent = pgno
}

func (p *MemPage) full(data interface{}) bool {
  switch data.(type){
  case *Cell:
    if p.ph.flag == INTERPAGE {
      return p.ph.nFree > (pl.size + size(Cell))
    }
    panic("full error")
  case *PlayLoad:
    if p.ph.flag == LEAFPAGE {
      return p.ph.nFree > (pl.size + size(Cell))
    }
    panic("full error")
  }
}
